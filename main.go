package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/golang/glog"
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

const (
	numThreads = 1 // TODO This will be a cli flag or be removed as a concept
)

func main() {
	// parse flags
	flag.Parse()
	err := flag.Lookup("logtostderr").Value.Set("true") // Glog is weird in that by default it logs to a file. Change it so that by default it all goes to stderr. (no option for stdout).
	if err != nil {
		glog.Fatal("Error setting default flag: ", err)
	}
	defer glog.Flush() // This should ensure that everything makes it out on to the console if the program crashes.

	glog.Info("Starting Data Collector")

	// Create in/out channels for the transformer
	transformInput := make(chan *unstructured.Unstructured)
	transformOutput := make(chan transforms.Node)
	t := transforms.Transformer{
		Input:  transformInput,
		Output: transformOutput,
	}

	// Start the transformer with 1 threads doing transformation work.
	glog.Infof("Starting %d transformer threads", numThreads)
	err = t.Start(numThreads)
	if err != nil {
		panic(err)
	}

	var config *rest.Config
	var clientConfigError error

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		glog.Info("Creating k8s client using config from KUBECONFIG")
		config, clientConfigError = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".kube", "config")); os.IsNotExist(err) {
		glog.Info("Creating k8s client using InClusterConfig()")
		config, clientConfigError = rest.InClusterConfig()
	} else {
		glog.Info("Creating k8s client using config from ~/.kube/config")
		config, clientConfigError = clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	}

	if clientConfigError != nil {
		glog.Fatal("Error Constructing Client From Config: ", clientConfigError)
	}

	// Initialize the dynamic client, used for CRUD operations on nondeafult k8s resources
	dynamicClientset, err := dynamic.NewForConfig(config)
	if err != nil {
		glog.Fatal("Cannot Construct Dynamic Client From Config: ", err)
	}

	// Create informer factories
	dynamicFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClientset, 0) // factory for building dynamic informer objects used with CRDs and arbitrary k8s objects

	// Create special type of client used for discovering resource types
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
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
				t.Input <- resource // Send resource into the transformer input channel
			},
		})

		go informer.Run(stopper)
	}

	receiver(transformOutput) // Would be the sender. For now, just a simple function to print it for testing.
}

// Basic receiver that prints stuff, for my testing
func receiver(transformOutput chan transforms.Node) {
	glog.Info("Receiver started") //RM
	for {
		n := <-transformOutput
		name, ok := n.Properties["name"].(string)
		if !ok {
			name = "UNKNOWN"
		}

		kind, ok := n.Properties["kind"].(string)
		if !ok {
			kind = "UNKNOWN"
		}
		glog.Info("Received: " + kind + " " + name)
	}
}
