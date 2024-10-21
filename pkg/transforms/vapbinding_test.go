// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
func TestVapBinding(t *testing.T) {
	var object map[string]interface{}
	UnmarshalFile("vapbinding.json", &object, t)

	unstructured := &unstructured.Unstructured{
		Object: object,
	}

	node := VapBindingResourceBuilder(unstructured).BuildNode()

	AssertDeepEqual("validationActions", node.Properties["validationActions"], []string{"Deny", "Warn", "Audit"}, t)
	AssertEqual("_ownedByGatekeeper", node.Properties["_ownedByGatekeeper"], true, t)
	AssertEqual("policyName", node.Properties["policyName"], "demo-policy.example.com", t)
}
