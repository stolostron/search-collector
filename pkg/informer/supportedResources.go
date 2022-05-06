// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"context"

	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/config"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
)

type Resource struct {
	ApiGroups []string `yaml:"apiGroups"`
	Resources []string `yaml:"resources"`
}

func GetAllowDenyData(cm *v1.ConfigMap) ([]Resource, []Resource, error, error) {

	var allow []Resource
	allowerr := yaml.Unmarshal([]byte(cm.Data["AllowedResources"]), &allow)
	if allowerr != nil {
		glog.Errorf(`Error while parsing allowed resources from ConfigMap. 
		Can't use configured value, defaulting to allow all resources. %v`, allowerr)
	}

	var deny []Resource
	denyerr := yaml.Unmarshal([]byte(cm.Data["DeniedResources"]), &deny)
	if denyerr != nil {
		glog.Errorf(`Error while parsing allowed resources from ConfigMap. 
		Can't use configured value, defaulting to deny all resources. %v`, denyerr)
	}

	return allow, deny, allowerr, denyerr
}

func isResourceAllowed(group, kind string, allowedList []Resource, deniedList []Resource) bool {

	// Ignore clusters and clusterstatus resources because these are handled by the aggregator.
	// Ignore oauthaccesstoken resources because those cause too much noise on OpenShift clusters.
	// Ignore projects as namespaces are overwritten to be projects on Openshift clusters - they tend to share
	// the same uid.
	list := []string{"events", "projects", "clusters", "clusterstatuses", "oauthaccesstokens"}

	// Deny all apiResources with kind in list
	for _, name := range list {
		if kind == name {
			glog.V(2).Infof("Deny resource [group: '%s' kind: %s]. Search collector doesn't support it.", group, kind)
			return false
		}
	}

	// Deny resources that match the deny list.
	g, k, denied := isResourceMatchingList(deniedList, group, kind)
	if denied {

		// Check if resource is also in the allow list.
		_, _, allowed := isResourceMatchingList(allowedList, group, kind)
		if allowed {
			glog.V(2).Infof("Deny Resource [group: '%s' kind: %s]. Resource present in both allow and deny rule.", group, kind)
		} else {
			glog.V(2).Infof("Deny resource [group: '%s' kind: %s]. Matched rule [group: '%s' kind: %s].", group, kind, g, k)
		}
		return false
	}

	// If allowList not provided, interpret it as allow all resources.
	// otherwise allow only the resources declared in allow list.
	if len(allowedList) == 0 {
		glog.V(2).Infof("Allow resource [group: '%s' kind: %s]. AllowList is empty.", group, kind)
		return true
	} else {
		g, k, allowed := isResourceMatchingList(allowedList, group, kind)
		if allowed {
			glog.V(2).Infof("Allow resource [group: '%s' kind: %s]. Matched [group: '%s' kind: %s].", group, kind, g, k)
			return true
		}
	}

	glog.V(2).Infof("Deny resource [group: '%s' kind: %s]. It doesn't match any allow or deny rule.", group, kind)
	return false
}

//helper funtion to get map of resource object
func isResourceMatchingList(resourceList []Resource, group, kind string) (string, string, bool) {
	for _, r := range resourceList {
		for _, g := range r.ApiGroups {
			for _, k := range r.Resources {
				if (g == "*" || g == group) && (k == "*" || k == kind) { // Group and kind matches
					return g, k, true
				}
			}
		}
	}
	return "", "", false
}

// Returns a map containing all the GVRs on the cluster of resources that support WATCH (ignoring clusters and events).
func SupportedResources(discoveryClient *discovery.DiscoveryClient) (map[schema.GroupVersionResource]struct{}, error) {
	// Next step is to discover all the gettable resource types that the kuberenetes api server knows about.
	supportedResources := []*machineryV1.APIResourceList{}

	// List out all the preferred api-resources of this server.
	apiResources, err := discoveryClient.ServerPreferredResources() // here we get preferred api versions
	if err != nil && apiResources == nil {                          // only return if the list is empty
		return nil, err
	} else if err != nil {
		glog.Warning("ServerPreferredResources could not list all available resources: ", err)
	}

	// create client to get configmap
	kubeClient := config.GetKubeClient(config.GetKubeConfig())

	// locate the search-collector-config ConfigMap
	cm, err2 := kubeClient.CoreV1().ConfigMaps(config.Cfg.PodNamespace).
		Get(context.TODO(), "search-collector-config", metav1.GetOptions{})
	if err2 != nil {
		glog.Info("Didn't find ConfigMap with name search-collector-config. Will collect all resources. ", err2)
	}

	// parse alloy/deny from config
	allowedList, deniedList, _, _ := GetAllowDenyData(cm)

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
