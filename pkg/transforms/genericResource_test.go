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
	expectedAnnotationKeys := sets.New(
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

func Test_edgesFromVirtualMachineInstanceMigration(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineinstancemigration.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	nodes := []Node{
		{UID: "uuid-123-vmim", Properties: map[string]interface{}{"kind": "VirtualMachineInstance", "namespace": "ugo", "name": "rhel-10-crimson-eagle-72"}},
	}
	nodeStore := BuildFakeNodeStore(nodes)

	edges := make([]Edge, 0)
	edges = edgesByDefaultTransformConfig(edges, node, nodeStore)

	AssertEqual("VMI edge total: ", len(edges), 1, t)
	AssertEqual("VMI migrationOf", edges[0].DestKind, "VirtualMachineInstance", t)
}

func Test_edgesFromVirtualMachine(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachine.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	nodes := []Node{
		{UID: "uuid-123-pvc-1", Properties: map[string]interface{}{"kind": "PersistentVolumeClaim", "namespace": "openshift-cnv", "name": "the-claim-is-persistent"}},
		{UID: "uuid-123-pvc-2", Properties: map[string]interface{}{"kind": "PersistentVolumeClaim", "namespace": "openshift-cnv", "name": "the-claim-is-too-persistent"}},
		{UID: "uuid-123-dv-1", Properties: map[string]interface{}{"kind": "DataVolume", "namespace": "openshift-cnv", "name": "rhel-8-amber-fish-51-volume"}},
		{UID: "uuid-123-dv-2", Properties: map[string]interface{}{"kind": "DataVolume", "namespace": "openshift-cnv", "name": "rhel-8-amber-fish-51-volume-2"}},
	}
	nodeStore := BuildFakeNodeStore(nodes)

	edges := make([]Edge, 0)
	edges = edgesByDefaultTransformConfig(edges, node, nodeStore)

	AssertEqual("VM edge total: ", len(edges), 4, t)
	AssertEqual("VM attachedTo", edges[0].DestKind, "DataVolume", t)
	AssertEqual("VM attachedTo dv name: ", edges[0].DestUID, "uuid-123-dv-1", t)
	AssertEqual("VM attachedTo", edges[1].DestKind, "DataVolume", t)
	AssertEqual("VM attachedTo dv name: ", edges[1].DestUID, "uuid-123-dv-2", t)
	AssertEqual("VM attachedTo", edges[2].DestKind, "PersistentVolumeClaim", t)
	AssertEqual("VM attachedTo pvc name: ", edges[2].DestUID, "uuid-123-pvc-1", t)
	AssertEqual("VM attachedTo", edges[3].DestKind, "PersistentVolumeClaim", t)
	AssertEqual("VM attachedTo pvc name: ", edges[3].DestUID, "uuid-123-pvc-2", t)
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
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"AgentConnected":   "True",
		"DataVolumesReady": "True",
		"Initialized":      "True",
		"LiveMigratable":   "False",
		"Ready":            "True",
	}, t)
	AssertEqual("cpu", node.Properties["cpu"], int64(1), t)
	AssertEqual("_description", node.Properties["_description"], "some description", t)
	AssertEqual("flavor", node.Properties["flavor"], "small", t)
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("osName", node.Properties["osName"], "rhel9", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("runStrategy", node.Properties["runStrategy"], nil, t)
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
	AssertEqual("cpu", node.Properties["cpu"], int64(1), t)
	AssertEqual("ipaddress", node.Properties["ipaddress"], "10.128.1.193", t)
	AssertEqual("liveMigratable", node.Properties["liveMigratable"], "False", t)
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("node", node.Properties["node"], "sno-0-0", t)
	AssertEqual("osVersion", node.Properties["osVersion"], "7 (Core)", t)
	AssertEqual("phase", node.Properties["phase"], "Running", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("vmSize", node.Properties["vmSize"], "small", t)
}

func Test_genericResourceFromConfigVMIM(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineinstancemigration.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("created", node.Properties["created"], "2025-07-11T14:42:32Z", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineInstanceMigration", t)
	AssertDeepEqual("label", node.Properties["label"], map[string]string{"kubevirt.io/vm1-name": "rhel-10-crimson-eagle-72"}, t)
	AssertEqual("name", node.Properties["name"], "rhel-10-crimson-eagle-72-migration-j9h6b", t)
	AssertEqual("namespace", node.Properties["namespace"], "ugo", t)
	AssertEqual("vmiName", node.Properties["vmiName"], "rhel-10-crimson-eagle-72", t)

	// Verify properties defined in the transform config
	AssertEqual("endTime", node.Properties["endTime"], "2025-07-11T14:42:32Z", t)
	AssertEqual("phase", node.Properties["phase"], "Scheduling", t)
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
	AssertEqual("_conditionReadyReason", node.Properties["_conditionReadyReason"], "Operation complete", t)
	AssertEqual("phase", node.Properties["phase"], "Succeeded", t)
	AssertEqual("readyToUse", node.Properties["readyToUse"], true, t)
	AssertEqual("sourceName", node.Properties["sourceName"], "centos7-gray-owl-35", t)
	AssertEqual("sourceKind", node.Properties["sourceKind"], "VirtualMachine", t)
	AssertDeepEqual("indications", node.Properties["indications"], []interface{}{"Online", "NoGuestAgent"}, t)

}

func Test_genericResourceFromConfigVMRestore(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachinerestore.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos7-gray-owl-35-snapshot-20250506-102417-1746547073646", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineRestore", t)
	AssertEqual("namespace", node.Properties["namespace"], "openshift-cnv", t)
	AssertEqual("created", node.Properties["created"], "2024-09-18T19:43:53Z", t)

	// Verify properties defined in the transform config
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("_conditionReadyReason", node.Properties["_conditionReadyReason"], "Operation complete", t)
	AssertEqual("complete", node.Properties["complete"], true, t)
	AssertEqual("targetName", node.Properties["targetName"], "centos7-gray-owl-35", t)
	AssertEqual("targetKind", node.Properties["targetKind"], "VirtualMachine", t)
	AssertDeepEqual("restoreTime", node.Properties["restoreTime"], "2025-05-06T15:59:39Z", t)

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
