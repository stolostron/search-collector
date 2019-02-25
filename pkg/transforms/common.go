package transforms

import "k8s.io/apimachinery/pkg/apis/meta/v1"

type CommonNodeProperties struct {
	Uid             string            `json: _uid`
	ResourceVersion string            `json: _resourceVersion`
	Cluster         string            `json: cluster`
	Name            string            `json: name`
	Namespace       string            `json: namespace`
	SelfLink        string            `json: selfLink`
	Created         string            `json: created`
	Labels          map[string]string `json: labels`
}

func TransformCommon(resource v1.Object) CommonNodeProperties {
	return CommonNodeProperties{
		Uid:             string(resource.GetUID()),
		ResourceVersion: resource.GetResourceVersion(),
		Cluster:         resource.GetClusterName(),
		Name:            resource.GetName(),
		Namespace:       resource.GetNamespace(),
		SelfLink:        resource.GetSelfLink(),
		Created:         resource.GetCreationTimestamp().String(),
		Labels:          resource.GetLabels(),
	}
}
