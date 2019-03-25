package transforms

import (
	"testing"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

func TestTransformDeployable(t *testing.T) {
	var d mcm.Deployable
	UnmarshalFile("../../test-data/deployable.json", &d, t)
	node := transformDeployable(&d)

	// Test only the fields that exist in deployable - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Deployable", t)
	AssertEqual("deployerKind", node.Properties["deployerKind"], "helm", t)
	AssertEqual("chartUrl", node.Properties["chartUrl"], "https://awesomewebsite.com/test-0.1.0.tgz", t)
	AssertEqual("deployerNamespace", node.Properties["deployerNamespace"], "default", t)
}
