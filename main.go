/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package main

import (
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	rec "github.ibm.com/IBMPrivateCloud/search-collector/pkg/reconciler"
	tr "github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/send"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/tlsutil"
)

func main() {

	numThreads := runtime.NumCPU() // determine number of CPUs available. We make that many goroutines for transformation and reconciliation, so that we take maximum advantage of whatever hardware we're on
	glog.Info("Starting Data Collector")
	if commit, ok := os.LookupEnv("VCS_REF"); ok {
		glog.Info("Built from git commit: ", commit)
	}

	// Create Helm client for transformer
	var helmClient *helm.Client
	helmTlsConfig, err := tlsutil.ClientConfig(config.Cfg.TillerOpts)
	if err != nil {
		glog.Error("Error creating helm TLS configuration: ", err)
	} else {
		helmClient = helm.NewClient(
			helm.WithTLS(helmTlsConfig),
			helm.Host(config.Cfg.TillerURL),
		)
		glog.Info("Created new helm client")
	}

	// Create transformers
	upsertTransformer := tr.NewTransformer(make(chan *tr.Event), make(chan tr.NodeEvent), numThreads, helmClient)

	// Init reconciler
	reconciler := rec.NewReconciler()
	reconciler.Input = upsertTransformer.Output

	// Create Sender, attached to transformer
	sender := send.NewSender(reconciler, config.Cfg.AggregatorURL, config.Cfg.ClusterName)

	var clientConfig *rest.Config
	var clientConfigError error

	if config.Cfg.KubeConfig != "" {
		glog.Infof("Creating k8s client using path: %s", config.Cfg.KubeConfig)
		clientConfig, clientConfigError = clientcmd.BuildConfigFromFlags("", config.Cfg.KubeConfig)
	} else {
		glog.Info("Creating k8s client using InClusterlientConfig()")
		clientConfig, clientConfigError = rest.InClusterConfig()
	}

	if clientConfigError != nil {
		glog.Fatal("Error Constructing Client From Config: ", clientConfigError)
	}

	// Initialize the dynamic client, used for CRUD operations on arbitrary k8s resources
	dynamicClientset, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		glog.Fatal("Cannot Construct Dynamic Client From Config: ", err)
	}

	// Create informer factories
	dynamicFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClientset, 0) // factory for building dynamic informer objects used with CRDs and arbitrary k8s objects

	// Create special type of client used for discovering resource types
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
	if err != nil {
		glog.Fatal("Cannot Construct Discovery Client From Config: ", err)
	}

	// These functions return handler functions, which are then used in creation of the informers.
	createInformerAddHandler := func(resourceName string) func(interface{}) {
		return func(obj interface{}) {
			resource := obj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Create,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	createInformerUpdateHandler := func(resourceName string) func(interface{}, interface{}) {
		return func(oldObj, newObj interface{}) {
			resource := newObj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Update,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	informerDeleteHandler := func(obj interface{}) {
		resource := obj.(*unstructured.Unstructured)
		// We don't actually have anything to transform in the case of a deletion, so we manually construct the NodeEvent
		ne := tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Delete,
			Node: tr.Node{
				UID: strings.Join([]string{config.Cfg.ClusterName, string(resource.GetUID())}, "/"),
			},
		}
		reconciler.Input <- ne

		if tr.IsHelmRelease(resource) {
			releaseNE := tr.NodeEvent{
				Time:      time.Now().Unix(),
				Operation: tr.Delete,
				Node: tr.Node{
					UID: tr.GetHelmReleaseUID(resource.GetLabels()["NAME"]),
				},
			}
			reconciler.Input <- releaseNE
		}
	}

	stoppers := make(map[schema.GroupVersionResource]chan struct{}) // We keep each of the informer's stopper channel in a map, so we can stop them if the resource is no longer valid.
	// Start a routine to keep our informers up to date.
	go func() {
		for {
			gvrList, err := supportedResources(discoveryClient)
			if err != nil {
				glog.Error("Failed to get complete list of supported resources: ", err)
			}

			// Sometimes a partial list will be returned even if there is an error.
			// This could happen during install when a CRD hasn't fully initialized.
			if gvrList != nil {
				// Loop through the previous list of resources. If we find the entry in the new list we delete it so that we don't end up with 2 informers.
				// If we don't find it, we stop the informer that's currently running because the resource no longer exists (or no longer supports watch).
				for gvr, stopper := range stoppers {
					// If this still exists in the new list, delete it from there as we don't want to recreate an informer
					if _, ok := gvrList[gvr]; ok {
						delete(gvrList, gvr)
						continue
					} else { // if it's in the old and NOT in the new, stop the informer
						stopper <- struct{}{}
						glog.Infof("Resource %s no longer exists or no longer supports watch, stopping its informer\n", gvr.String())
						delete(stoppers, gvr)
					}
				}
				// Now, loop through the new list, which after the above deletions, contains only stuff that needs to have a new informer created for it.
				for gvr := range gvrList {
					// In this case we need to create a dynamic informer, since there is no built in informer for this type.
					dynamicInformer := dynamicFactory.ForResource(gvr)
					glog.Infof("Found new resource %s, creating informer\n", gvr.String())
					// Set up handler to pass this informer's resources into transformer
					informer := dynamicInformer.Informer()
					informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
						AddFunc:    createInformerAddHandler(gvr.Resource),
						UpdateFunc: createInformerUpdateHandler(gvr.Resource),
						DeleteFunc: informerDeleteHandler,
					})

					stopper := make(chan struct{})
					stoppers[gvr] = stopper
					go informer.Run(stopper)
				}
			}

			time.Sleep(time.Duration(config.Cfg.RediscoverRateMS) * time.Millisecond)
		}
	}()

	// Start a routine to send data every interval.
	backoffFactor := float64(0) // Used for exponential backoff, increased each interval. Has to be a float64 since I use it with math.Exp2()
	go func() {
		// First time send after 15 seconds, then send every ReportRateMS milliseconds.
		time.Sleep(15 * time.Second)
		for {
			glog.V(2).Info("Beginning Send Cycle")
			err = sender.Sync()
			if err != nil {
				glog.Error("SENDING ERROR: ", err)
				if time.Duration(config.Cfg.ReportRateMS)*time.Duration(math.Exp2(backoffFactor))*time.Millisecond < time.Duration(config.Cfg.MaxBackoffMS)*time.Millisecond {
					backoffFactor++ // Increase the backoffFactor, doubling the wait time. Stops doubling it after it passes the max wait time (an hour) so that we don't overflow int.
				}
			} else {
				glog.V(2).Info("Send Cycle Completed Successfully")
				backoffFactor = float64(0)
			}
			timeToSleep := time.Duration(min(config.Cfg.ReportRateMS*int(math.Exp2(backoffFactor)), config.Cfg.MaxBackoffMS)) * time.Millisecond
			if backoffFactor > 0 {
				glog.Warning("Backing off send interval because of error response from aggregator. Sleeping for ", timeToSleep)
			}
			time.Sleep(timeToSleep) // Sleep either for the current backed off interval, or the maximum time defined in the config
		}
	}()

	// We don't actually use this to wait on anything, since the transformer routines don't ever end unless something goes wrong. We just use this to wait forever in main once we start things up.
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait() // This will never end (until we kill the process)
}

// Returns the smaller of two ints
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// Returns a map containing all the GVRs on the cluster of resources that support WATCH (ignoring clusters and events).
func supportedResources(discoveryClient *discovery.DiscoveryClient) (map[schema.GroupVersionResource]struct{}, error) {
	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil && apiResources == nil { // only return if the list is empty
		return nil, err
	} else if err != nil {
		glog.Warning("ServerPreferredResources could not list all available resources: ", err)
	}

	// Filter down to only resources which support WATCH operations.
	for _, apiList := range apiResources { // This comes out in a nested list, so loop through a couple things
		watchList := machineryV1.APIResourceList{} // This is a copy of apiList but we only insert resources for which GET is supported.
		watchList.GroupVersion = apiList.GroupVersion
		watchResources := []machineryV1.APIResource{}      // All the resources for which GET works.
		for _, apiResource := range apiList.APIResources { // Loop across inner list
			// TODO: Use env variable for ignored resource kinds.
			// Ignore clusters and clusterstatus resources because these are handled by the aggregator.
			// Ignore events because those cause too much noise.
			// Ignore oauthaccesstoken resources because those cause too much noise on OpenShift clusters.
			if apiResource.Name == "clusters" || apiResource.Name == "clusterstatuses" || apiResource.Name == "events" || apiResource.Name == "oauthaccesstoken" {
				continue
			}
			for _, verb := range apiResource.Verbs {
				if verb == "watch" {
					watchResources = append(watchResources, apiResource)
				}
			}
		}
		watchList.APIResources = watchResources
		supportedResources = append(supportedResources, &watchList) // Add the list to our list of lists that holds GET enabled resources.
	}

	// Use handy converter function to convert into GroupVersionResource objects, which we need in order to make informers
	gvrList, err := discovery.GroupVersionResources(supportedResources)

	return gvrList, err
}
