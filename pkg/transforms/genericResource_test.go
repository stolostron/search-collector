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
	AssertEqual("agentConnected", node.Properties["agentConnected"], "True", t)
	AssertEqual("cpu", node.Properties["cpu"], int64(1), t)
	AssertEqual("flavor", node.Properties["flavor"], "small", t)
	AssertEqual("memory", node.Properties["memory"], "2Gi", t)
	AssertEqual("osName", node.Properties["osName"], "rhel9", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("status", node.Properties["status"], "Running", t)
	AssertEqual("workload", node.Properties["workload"], "server", t)
	AssertEqual("_specRunning", node.Properties["_specRunning"], true, t)
	AssertEqual("_specRunStrategy", node.Properties["_specRunStrategy"], nil, t)
}

func Test_genericResourceFromConfigVMI(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineinstance.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos7-gray-owl-35", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineInstance", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-09-18T19:43:53Z", t)

	// Verify properties defined in the transform config
	AssertEqual("ipaddress", node.Properties["ipaddress"], "10.128.1.193", t)
	AssertEqual("liveMigratable", node.Properties["liveMigratable"], "False", t)
	AssertEqual("node", node.Properties["node"], "sno-0-0", t)
	AssertEqual("osVersion", node.Properties["osVersion"], "7 (Core)", t)
	AssertEqual("phase", node.Properties["phase"], "Running", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("vmSize", node.Properties["vmSize"], "small", t)

}

func Test_genericResourceFromConfigVMSnapshot(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachinesnapshot.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos7-gray-owl-35-snapshot", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineSnapshot", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-09-18T19:43:53Z", t)

	// Verify properties defined in the transform config
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("status", node.Properties["status"], "Operation complete", t)
	AssertEqual("sourceVM", node.Properties["sourceVM"], "centos7-gray-owl-35", t)
	AssertDeepEqual("accesindicationssMode", node.Properties["indications"], []interface{}{"Online", "NoGuestAgent"}, t)

}

func Test_genericResourceFromConfigDataVolume(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("datavolume.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos7-gray-owl-35", t)
	AssertEqual("kind", node.Properties["kind"], "DataVolume", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-09-09T20:00:42Z", t)

	// Verify properties defined in the transform config
	AssertEqual("size", node.Properties["size"], "20Gi", t)
	AssertEqual("storageClassName", node.Properties["storageClassName"], nil, t)
}

func Test_genericResourceFromConfigNamespace(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("namespace.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "default", t)
	AssertEqual("kind", node.Properties["kind"], "Namespace", t)
	AssertEqual("namespace", node.Properties["namespace"], nil, t)
	AssertEqual("created", node.Properties["created"], "2019-02-21T21:25:42Z", t)

	// Verify properties defined in the transform config
	AssertEqual("status", node.Properties["status"], "Active", t)
}

func Test_genericResourceFromConfigStorageClass(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("storageclass.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "gp2-csi", t)
	AssertEqual("kind", node.Properties["kind"], "StorageClass", t)
	AssertEqual("namespace", node.Properties["namespace"], nil, t)
	AssertEqual("created", node.Properties["created"], "2025-03-11T10:24:44Z", t)

	// Verify properties defined in the transform config
	AssertEqual("allowVolumeExpansion", node.Properties["allowVolumeExpansion"], true, t)
	AssertEqual("provisioner", node.Properties["provisioner"], "ebs.csi.aws.com", t)
	AssertEqual("reclaimPolicy", node.Properties["reclaimPolicy"], "Delete", t)
	AssertEqual("volumeBindingMode", node.Properties["volumeBindingMode"], "WaitForFirstConsumer", t)
}
