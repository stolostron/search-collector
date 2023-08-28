// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

// Declares a property to extract from a resource using a json path.
type ExtractProperty struct {
	name     string   // `json:"name,omitempty"`
	propType string   // `json:"type,omitempty"`
	path     []string // `json:"path,omitempty"`
}

// Declares properties to extract from a given resource.
type ResourceConfig struct {
	// apigroup   string                         // `json:"apigroup,omitempty"`
	// kind       string                         // `json:"kind,omitempty"`
	properties []ExtractProperty // `json:"properties,omitempty"`
}

// ConfigurableResourceBuilder ...
func ConfigurableResourceBuilder(r *unstructured.Unstructured, kind, group string) *GenericResource {
	n := Node{
		UID:        prefixedUID(r.GetUID()),
		Properties: unstructuredProperties(r),
		Metadata:   unstructuredMetadata(r),
	}

	transformConfig := getTransformConfig(kind, group)

	for _, prop := range transformConfig.properties {
		if prop.propType != "string" {
			klog.Errorf("Property %s has unsupported type %s", prop.name, prop.propType)
			continue
		}

		switch prop.propType {
		case "string":
			val, found, err := unstructured.NestedString(r.Object, prop.path...)
			if err != nil {
				klog.Errorf("Error extracting property %s from resource %s: %v", prop.name, r.GetName(), err)
				continue
			} else if !found {
				klog.Errorf("Property %s not found in resource %s", prop.name, r.GetName())
				continue
			}
			n.Properties[prop.name] = val

		default:
			klog.Errorf("Property %s has unsupported type %s", prop.name, prop.propType)
			continue
		}
	}

	klog.Infof("Built configurable resource [%s.%s] node... %+v\n\n", kind, group, n)

	return &GenericResource{node: n}
}

// BuildNode()
// Implemented by GenericResourceBuilder().

// BuildEdges
// Implemented by GenericResourceBuilder().
