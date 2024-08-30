// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

func Test_genericResourceFromConfig(t *testing.T) {
	config.Cfg.CollectAnnotations = true

	defer func() {
		config.Cfg.CollectAnnotations = false
	}()

	var r unstructured.Unstructured
	UnmarshalFile("clusterserviceversion.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "advanced-cluster-management.v2.9.0", t)
	AssertEqual("kind", node.Properties["kind"], "ClusterServiceVersion", t)
	AssertEqual("namespace", node.Properties["namespace"], "open-cluster-management", t)
	AssertEqual("created", node.Properties["created"], "2023-08-23T15:54:22Z", t)

	annotations, ok := node.Properties["annotation"].(map[string]string)
	assert.True(t, ok)

	// Ensure last-applied-configuration and other large annotations are not present
	expectedAnnotationKeys := sets.New[string](
		"capabilities", "categories", "certified", "createdAt", "olm.operatorGroup",
		"olm.operatorNamespace", "olm.targetNamespaces", "operatorframework.io/suggested-namespace",
		"operators.openshift.io/infrastructure-features", "operators.operatorframework.io/internal-objects", "support",
	)

	actualAnnotationKeys := sets.Set[string]{}

	for key := range annotations {
		actualAnnotationKeys.Insert(key)
	}

	assert.True(t, expectedAnnotationKeys.Equal(actualAnnotationKeys))

	// Verify properties defined in the transform config
	AssertEqual("display", node.Properties["display"], "Advanced Cluster Management for Kubernetes", t)
	AssertEqual("phase", node.Properties["phase"], "Succeeded", t)
	AssertEqual("version", node.Properties["version"], "2.9.0", t)

	// Verify that annotations are not collected when COLLECT_ANNOTATIONS is false
	config.Cfg.CollectAnnotations = false

	node = GenericResourceBuilder(&r).BuildNode()
	assert.Nil(t, node.Properties["annotations"])
}

func Test_allowListedForAnnotations(t *testing.T) {
	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: POLICY_OPEN_CLUSTER_MANAGEMENT_IO, Kind: "Policy", Version: "v1",
	})
	obj.SetAnnotations(map[string]string{"hello": "world"})

	node := GenericResourceBuilder(&obj).BuildNode()
	assert.NotNil(t, node.Properties["annotation"])

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "constraints.gatekeeper.sh", Kind: "K8sRequiredLabels", Version: "v1beta1",
	})

	node = GenericResourceBuilder(&obj).BuildNode()
	assert.NotNil(t, node.Properties["annotation"])

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "something.domain.example", Kind: "SomeKind", Version: "v1",
	})

	node = GenericResourceBuilder(&obj).BuildNode()
	assert.Nil(t, node.Properties["annotation"])
}

func Test_genericResourceFromConfigVM(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachine.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "rhel9-gitops", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachine", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-04-30T16:22:02Z", t)

	// Verify properties defined in the transform config
	AssertEqual("display", node.Properties["status"], "Running", t)
	AssertEqual("phase", node.Properties["ready"], "True", t)

}
