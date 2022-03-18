package informer

import (
	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/config"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

type Resource struct {
	ApiGroups []string `yaml:"apiGroups"`
	Resources []string `yaml:"resources"`
}

func GetAllowDenyData(cm *v1.ConfigMap) ([]Resource, []Resource) {

	var allow []Resource
	allowerr := yaml.Unmarshal([]byte(cm.Data["AllowedResources"]), &allow)
	if allowerr != nil {
		klog.Errorf("Error while parsing allowed resources from ConfigMap. Can't use configured value, defaulting to allow all resources. %v", allowerr)
	}

	var deny []Resource
	denyerr := yaml.Unmarshal([]byte(cm.Data["DeniedResources"]), &deny)
	if denyerr != nil {
		klog.Errorf("Error while parsing allowed resources from ConfigMap. Can't use configured value, defaulting to deny all resources. %v", denyerr)
	}

	return allow, deny
}

func isResourceAllowed(group, kind string, allowedList []Resource, deniedList []Resource) bool {

	// Ignore clusters and clusterstatus resources because these are handled by the aggregator.
	// Ignore oauthaccesstoken resources because those cause too much noise on OpenShift clusters.
	// Ignore projects as namespaces are overwritten to be projects on Openshift clusters - they tend to share
	// the same uid.
	list := []string{"events", "projects", "clusters", "clusterstatuses", "oauthaccesstokens"}

	//remove all apiResources with kind in list
	for _, name := range list {
		if kind == name {
			return false
		}
	}

	// remove denied resources from all apigroups if * otherwise remove from specific apigroups.
	for _, deny := range deniedList {
		for _, api := range deny.ApiGroups {
			for _, rec := range deny.Resources {

				if api == "*" && rec != "*" && rec == kind {
					klog.V(1).Infof("Resource %s %s matched deny list. ", group, kind)
					return false
				} else if api != "*" && rec != "*" && rec == kind && group == api {
					klog.V(1).Infof("Resource %s %s matched deny list. ", group, kind)
					return false
				} else if api != "*" && rec == "*" && group == api {
					klog.V(1).Infof("Resource %s %s matched deny list. ", group, kind)
					return false
				}
			}

		}

	}

	// if allowedlist is empty allow all resources, otherwise if * allow all groups specific
	// resources if not * allow specific resources to specific groups:
	if len(allowedList) == 0 {
		return true
	} else {
		for _, allow := range allowedList {
			for _, api := range allow.ApiGroups {
				for _, rec := range allow.Resources {

					if api == "*" && rec != "*" && rec == kind { //all api, specific resources
						klog.V(1).Infof("Resource %s %s matched allow list. ", group, kind)
						return true
					} else if api != "*" && rec != "*" && rec == kind && group == api { //specific api, specific resources
						klog.V(1).Infof("Resource %s %s matched allow list. ", group, kind)
						return true
					} else if api != "*" && rec == "*" && group == api { //specific api, all resoruces
						klog.V(1).Infof("Resource %s %s matched allow list. ", group, kind)
						return true
					}
					if kind != rec && group != api {
						return false
					}
				}

			}

		}

	}

	klog.Warningf("Resource %s %s missing case.", kind, group)

	return true
}

// Returns a map containing all the GVRs on the cluster of resources that support WATCH (ignoring clusters and events).
func SupportedResources(discoveryClient *discovery.DiscoveryClient) (map[schema.GroupVersionResource]struct{}, error) {
	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources() //<--here we can look into this list and have preferred versions
	if err != nil && apiResources == nil {                          // only return if the list is empty
		return nil, err
	} else if err != nil {
		glog.Warning("ServerPreferredResources could not list all available resources: ", err)
	}

	// create client to get configmap
	kubeClient := config.GetKubeConfig() //can't err here?
	clientset, err := kubernetes.NewForConfig(kubeClient)
	if err != nil {
		glog.Info("Error when trying to create a clientset", err)
	}

	//locate the allow-deny ConfigMap:
	cm, err2 := clientset.CoreV1().ConfigMaps(config.Cfg.PodNamespace).Get(contextVar, "search-collector-config", metav1.GetOptions{})
	if err2 != nil {
		glog.Info("Didn't find ConfigMap with name search-collector-config. Will collect all resources. ", err2)
	}

	//parse config:
	allowedList, deniedList := GetAllowDenyData(cm)

	tr.NonNSResourceMap = make(map[string]struct{}) //map to store non-namespaced resources

	// Filter down to only resources which support WATCH operations
	for _, apiList := range apiResources { // This comes out in a nested list, so loop through a couple things
		// This is a copy of apiList but we only insert resources for which GET is supported.
		watchList := machineryV1.APIResourceList{}
		watchList.GroupVersion = apiList.GroupVersion
		watchResources := []machineryV1.APIResource{} // All the resources for which GET works.

		for _, apiResource := range apiList.APIResources { // Loop across inner list

			if !isResourceAllowed(apiResource.Group, apiResource.Name, allowedList, deniedList) {
				continue // Skip the resource before starting the informer
			}

			// add non-namespaced resource to NonNSResourceMap
			if !apiResource.Namespaced {
				tr.NonNSResMapMutex.Lock()
				if _, ok := tr.NonNSResourceMap[apiResource.Kind]; !ok {
					tr.NonNSResourceMap[apiResource.Kind] = struct{}{}
				}
				tr.NonNSResMapMutex.Unlock()

			}
			for _, verb := range apiResource.Verbs {
				if verb == "watch" {
					watchResources = append(watchResources, apiResource)
				}
			}
		}

		watchList.APIResources = watchResources
		supportedResources = append(supportedResources, &watchList)

	}

	// Use handy converter function to convert into GroupVersionResource objects, which we need in order to make informers
	gvrList, err := discovery.GroupVersionResources(supportedResources)

	return gvrList, err
}
