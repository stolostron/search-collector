package main

import (
	"runtime"
	"sync"
	"time"

	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/send"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	numThreads := runtime.NumCPU() // determine number of CPUs available. We make that many goroutines for transformation and reconciliation, so that we take maximum advantage of whatever hardware we're on
	glog.Info("Starting Data Collector")

	// Create transformers
	upsertTransformer := transforms.NewTransformer(make(chan *transforms.Event), make(chan transforms.NodeEvent), numThreads)

	// Create Sender, attached to transformer
	sender := send.NewSender(upsertTransformer.Output, config.Cfg.AggregatorURL, config.Cfg.ClusterName, numThreads)

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

	// Initialize the dynamic client, used for CRUD operations on nondeafult k8s resources
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

	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		glog.Fatal("Cannot list supported resources on k8s api-server: ", err)
	}

	// Filter down to only resources which support WATCH operations.
	for _, apiList := range apiResources { // This comes out in a nested list, so loop through a couple things
		watchList := machineryV1.APIResourceList{} // This is a copy of apiList but we only insert resources for which GET is supported.
		watchList.GroupVersion = apiList.GroupVersion
		watchResources := []machineryV1.APIResource{}      // All the resources for which GET works.
		for _, apiResource := range apiList.APIResources { // Loop across inner list
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

	if err != nil {
		glog.Fatal("Could not read api-resource object", err) // TODO pretty sure this would be fatal but I don't actually know how to produce it, so... we'll see! :)
	}

	stopper := make(chan struct{}) // We just have one stopper channel that we pass to all the informers, we always stop them together.
	defer close(stopper)

	// Now that we have a list of all the GVRs for resources we support, make dynamic informers for each one, set them up to pass their data into the transformer, and start them.
	for gvr := range gvrList {
		// Ignore events because those cause too much noise.
		// Ignore clusters and clusterstatus resources because these are handled by the aggregator.
		if gvr.Resource == "clusters" || gvr.Resource == "clusterstatuses" || gvr.Resource == "events" {
			continue
		}

		// In this case we need to create a dynamic informer, since there is no built in informer for this type.
		dynamicInformer := dynamicFactory.ForResource(gvr)
		glog.Infof("Created informer for %s \n", gvr.String())
		// Set up handler to pass this informer's resources into transformer
		informer := dynamicInformer.Informer()
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				resource := obj.(*unstructured.Unstructured)
				upsert := transforms.Event{
					Time:      time.Now().Unix(),
					Operation: transforms.Create,
					Resource:  resource,
				}
				upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				resource := newObj.(*unstructured.Unstructured)
				upsert := transforms.Event{
					Time:      time.Now().Unix(),
					Operation: transforms.Update,
					Resource:  resource,
				}
				upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
			},
			DeleteFunc: func(obj interface{}) {
				resource := obj.(*unstructured.Unstructured)
				uid := string(resource.GetUID())
				// We don't actually have anything to transform in the case of a deletion, so we manually construct the NodeEvent
				ne := transforms.NodeEvent{
					Time: time.Now().Unix(),
					Node: transforms.Node{
						UID: uid,
					},
				}
				sender.InputChannel <- ne
			},
		})

		go informer.Run(stopper)
	}

	//TODO make this a lot more robust, handle diffs, etc.
	// Start a really basic sender routine.
	go func() {
		// First time send after 10 seconds, then send every 5 seconds.
		time.Sleep(10 * time.Second)
		for {
			glog.Info("Beginning Send Cycle")
			err = sender.Sync()
			if err != nil {
				glog.Error("SENDING ERROR: ", err)
			} else {
				glog.Info("Send Cycle Completed Successfully")
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// We don't actually use this to wait on anything, since the transformer routines don't ever end unless something goes wrong. We just use this to wait forever in main once we start things up.
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait() // This will never end (until we kill the process)
}
