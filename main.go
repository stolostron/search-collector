package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1" // This one has the interface
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Create in/out channels for the transformer
	transformInput := make(chan machineryV1.Object)
	transformDynamicInput := make(chan *unstructured.Unstructured)
	transformOutput := make(chan transforms.Node)
	t := transforms.Transformer{
		Input:        transformInput,
		DynamicInput: transformDynamicInput,
		Output:       transformOutput,
	}

	// Start the transformer with 1 threads doing transformation work.
	err := t.Start(1)
	if err != nil {
		panic(err)
	}

	kubeconfig := os.Getenv("KUBECONFIG")

	if home := os.Getenv("HOME"); kubeconfig == "" && home != "" {
		log.Println("KUBECONFIG was undefined, using ~/.kube/config")
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Initialize the normal client, used for CRUD operations on default k8s resources
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Initialize the dynamic client, used for CRUD operations on nondeafult k8s resources
	dynamicClientset, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Create informer factories
	factory := informers.NewSharedInformerFactory(clientset, 0)                            // factory for building informer objects used with default k8s resources
	dynamicFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClientset, 0) // factory for building dynamic informer objects used with CRDs and arbitrary k8s objects

	// Create special type of client used for discovering resource types
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		panic(err.Error())
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
		panic(err.Error())
	}

	stopper := make(chan struct{}) // We just have one stopper channel that we pass to all the informers, we always stop them together.
	defer close(stopper)

	fmt.Printf("Supported resource types: %d\n", len(gvrList))
	// Create informers for every supported resource type.
	for gvr := range gvrList {
		genericInformer, err := factory.ForResource(gvr) // Attempt to create standard informer. This will return an erro on non-default k8s resources.
		var informer cache.SharedIndexInformer
		if err != nil {
			if isNoInformerError(err) { // This will be true when the resource is nondefault
				// In this case we need to create a dynamic informer, since there is no built in informer for this type.
				dynamicInformer := dynamicFactory.ForResource(gvr)
				fmt.Printf("Created informer for %+v \n", gvr)
				// Set up handler to pass this informer's resources into transformer
				informer = dynamicInformer.Informer()
				informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						resource := obj.(*unstructured.Unstructured)
						t.DynamicInput <- resource // Send pod into the transformerChannel
					},
				})
				fmt.Printf("Created informer for %+v \n", gvr)

				go informer.Run(stopper)
			} else { // Some other error
				fmt.Printf("ERROR: Unable to create informer for %+v \n", gvr)
				fmt.Println(err)
			}
			// TODO check the error and split into 2 cases
			// TODO declare dynamic informer and start it
		} else {
			// TODO success case, run the informer
			// Set up handler to pass this informer's resources into transformer
			informer = genericInformer.Informer()

			informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					resource := obj.(machineryV1.Object)
					transformInput <- resource // Send pod into the transformerChannel
				},
			})
			informer := genericInformer.Informer()
			fmt.Printf("Created informer for %+v \n", gvr)

			go informer.Run(stopper)
		}

	}

	receiver(transformOutput) // Would be the sender. For now, just a simple function to print it for testing.
}

// Basic receiver that prints stuff, for my testing
func receiver(transformOutput chan transforms.Node) {
	fmt.Println("Receiver started") //RM
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
		fmt.Println("Received: " + kind + " " + name)
	}
}

// Checks whether the given error is a "no informer found" error
func isNoInformerError(e error) bool {
	return strings.HasPrefix(e.Error(), "no informer found for")
}
