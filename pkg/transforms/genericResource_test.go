// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	ocpapp "github.com/openshift/api/apps/v1"
	gvrpolicy "github.com/stolostron/governance-policy-propagator/api/v1"
	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	appDeployable "github.com/stolostron/multicloud-operators-deployable/pkg/apis/apps/v1"
	acmrule "github.com/stolostron/multicloud-operators-placementrule/pkg/apis/apps/v1"
	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	acmchannel "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
	acmhelmrelease "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/helmrelease/v1"
	acmapp "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/v1"
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
	AssertEqual("architecture", node.Properties["architecture"], "amd64", t)
	AssertEqual("agentConnected", node.Properties["agentConnected"], "True", t)
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"AgentConnected":   "True",
		"DataVolumesReady": "True",
		"Initialized":      "True",
		"LiveMigratable":   "False",
		"Ready":            "True",
	}, t)
	AssertEqual("cpu", node.Properties["cpu"], int64(1), t)
	AssertDeepEqual("dataVolumeNames", node.Properties["dataVolumeNames"],
		[]interface{}{"rhel-8-amber-fish-51-volume", "rhel-8-amber-fish-51-volume-2"}, t)
	AssertEqual("_description", node.Properties["_description"], "some description", t)
	AssertEqual("flavor", node.Properties["flavor"], "small", t)
	AssertDeepEqual("gpuName", node.Properties["gpuName"], []interface{}{"gpu-one", "gpu-two"}, t)
	AssertDeepEqual("hostDeviceName", node.Properties["hostDeviceName"], []interface{}{"host-device-one", "host-device-two"}, t)
	AssertEqual("instancetype", node.Properties["instancetype"], "instancetype-name", t)
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("osName", node.Properties["osName"], "rhel9", t)
	AssertEqual("preference", node.Properties["preference"], "preference-name", t)
	AssertDeepEqual("pvcClaimNames", node.Properties["pvcClaimNames"],
		[]interface{}{"the-claim-is-persistent", "the-claim-is-too-persistent"}, t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("runStrategy", node.Properties["runStrategy"], "always", t)
	AssertEqual("status", node.Properties["status"], "Running", t)
	AssertEqual("workload", node.Properties["workload"], "server", t)
	AssertEqual("_specRunning", node.Properties["_specRunning"], "true", t)
	AssertEqual("_specRunStrategy", node.Properties["_specRunStrategy"], "always", t)
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
	AssertEqual("cpuSockets", node.Properties["cpuSockets"], int64(1), t)
	AssertEqual("cpuThreads", node.Properties["cpuThreads"], int64(1), t)
	AssertEqual("guestOSInfoID", node.Properties["guestOSInfoID"], "centos", t)
	AssertDeepEqual("interfaceName", node.Properties["interfaceName"], []interface{}{"default", "default-2"}, t)
	AssertDeepEqual("_interface", node.Properties["_interface"], []string{
		"default/eth0[0]=10.128.1.193", "default/eth0[1]=fe80::60:ddff:fe00:4",
		"default2/eth0-2[0]=10.128.1.194", "default2/eth0-2[1]=fe80::60:ddff:fe00:5",
		"/eth0-2[0]=10.128.1.195", "/eth0-2[1]=fe80::60:ddff:fe00:6",
		"/[0]=10.128.1.196", "/[1]=fe80::60:ddff:fe00:7",
		"default3/[0]=10.128.1.197", "default3/[1]=fe80::60:ddff:fe00:8",
	}, t)
	AssertEqual("ipaddress", node.Properties["ipaddress"], "10.128.1.193", t)
	AssertEqual("liveMigratable", node.Properties["liveMigratable"], "False", t)
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("migrationPolicyName", node.Properties["migrationPolicyName"], "my-migration-policy", t)
	AssertEqual("node", node.Properties["node"], "sno-0-0", t)
	AssertEqual("osVersion", node.Properties["osVersion"], "7 (Core)", t)
	AssertEqual("phase", node.Properties["phase"], "Running", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("startStrategy", node.Properties["startStrategy"], "Paused", t)
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
	AssertEqual("deleted", node.Properties["deleted"], "2026-07-11T14:42:32Z", t)
	AssertEqual("endTime", node.Properties["endTime"], "2025-07-11T14:42:32Z", t)
	AssertEqual("migrationPolicyName", node.Properties["migrationPolicyName"], "my-first-migration-policy", t)
	AssertEqual("phase", node.Properties["phase"], "Scheduling", t)
	AssertEqual("sourceNode", node.Properties["sourceNode"], "node-1", t)
	AssertEqual("sourcePod", node.Properties["sourcePod"], "virt-launcher-rhel-10-crimson-eagle-72-zkzmn", t)
	AssertEqual("targetNode", node.Properties["targetNode"], "node-2", t)
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
	AssertEqual("readyToUse", node.Properties["readyToUse"], "true", t)
	AssertEqual("sourceName", node.Properties["sourceName"], "centos7-gray-owl-35", t)
	AssertEqual("sourceKind", node.Properties["sourceKind"], "VirtualMachine", t)
	AssertDeepEqual("indications", node.Properties["indications"], []interface{}{"Online", "NoGuestAgent"}, t)
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"Ready":       "True",
		"Progressing": "False",
	}, t)

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
	AssertEqual("complete", node.Properties["complete"], "true", t)
	AssertEqual("targetApiGroup", node.Properties["targetApiGroup"], "kubevirt.io", t)
	AssertEqual("targetName", node.Properties["targetName"], "centos7-gray-owl-35", t)
	AssertEqual("targetKind", node.Properties["targetKind"], "VirtualMachine", t)
	AssertDeepEqual("restoreTime", node.Properties["restoreTime"], "2025-05-06T15:59:39Z", t)
	AssertEqual("virtualMachineSnapshotName", node.Properties["virtualMachineSnapshotName"], "centos7-gray-owl-35-snapshot", t)

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
	AssertEqual("pvcName", node.Properties["pvcName"], "pvc-name", t)
	AssertEqual("pvcNamespace", node.Properties["pvcNamespace"], "pvc-namespace", t)
	AssertEqual("size", node.Properties["size"], "20Gi", t)
	AssertEqual("snapshotName", node.Properties["snapshotName"], "snapshot-name", t)
	AssertEqual("snapshotNamespace", node.Properties["snapshotNamespace"], "snapshot-namespace", t)
	AssertEqual("phase", node.Properties["phase"], "Succeeded", t)
	AssertEqual("storageClassName", node.Properties["storageClassName"], nil, t)
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"cdi.kubevirt.io/storage.usePopulator": "false",
	}, t)
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
	AssertEqual("allowVolumeExpansion", node.Properties["allowVolumeExpansion"], "true", t)
	AssertEqual("provisioner", node.Properties["provisioner"], "ebs.csi.aws.com", t)
	AssertEqual("reclaimPolicy", node.Properties["reclaimPolicy"], "Delete", t)
	AssertEqual("volumeBindingMode", node.Properties["volumeBindingMode"], "WaitForFirstConsumer", t)
}

func Test_genericResourceFromConfigWithMissingAttributes(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachine-missing-attributes.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify properties defined in the transform config, when not present on the resource being collected are not set
	AssertEqual("architecture", node.Properties["architecture"], nil, t)           // no architecture key on .spec.template.spec
	AssertEqual("agentConnected", node.Properties["agentConnected"], nil, t)       // empty conditions map on .status.conditions
	AssertDeepEqual("condition", node.Properties["condition"], nil, t)             // empty conditions map on .status.conditions
	AssertEqual("cpu", node.Properties["cpu"], nil, t)                             // no cpu map key on .spec.template.spec.domain
	AssertDeepEqual("dataVolumeNames", node.Properties["dataVolumeNames"], nil, t) // no dataVolume maps on .spec.template.spec.domain.volumes
	AssertEqual("_description", node.Properties["_description"], nil, t)           // no description key on .metadata.annotations
	AssertEqual("flavor", node.Properties["flavor"], nil, t)                       // no metadata map on .spec.template
	AssertEqual("memory", node.Properties["memory"], nil, t)                       // no memory map on .spec.template.spec.domain
	AssertEqual("osName", node.Properties["osName"], nil, t)                       // no metadata map on .spec.template
	AssertDeepEqual("pvcClaimNames", node.Properties["pvcClaimNames"], nil, t)     // no persistentVolumeClaim maps on .spec.template.spec.domain.volumes
	AssertEqual("ready", node.Properties["ready"], nil, t)                         // empty conditions map on .status.conditions
	AssertEqual("runStrategy", node.Properties["runStrategy"], nil, t)             // no runStrategy key on .spec
	AssertEqual("status", node.Properties["status"], nil, t)                       // no printableStatus key on .spec
	AssertEqual("workload", node.Properties["workload"], nil, t)                   // no metadata map on .spec.template
	AssertEqual("_specRunning", node.Properties["_specRunning"], nil, t)           // no running key on .spec
	AssertEqual("_specRunStrategy", node.Properties["_specRunStrategy"], nil, t)   // no runStrategy key on .spec
}

func Test_genericResourceFromConfigNetworkAddonsConfig(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("networkaddonsconfig.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "cluster", t)
	AssertEqual("kind", node.Properties["kind"], "NetworkAddonsConfig", t)
	AssertEqual("created", node.Properties["created"], "2025-12-01T14:05:35Z", t)

	// Verify status conditions
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"Degraded":    "False",
		"Progressing": "False",
		"Available":   "True",
	}, t)
}

func Test_genericResourceFromConfigVirtualMachineInstancetype(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineinstancetype.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "small", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineInstancetype", t)
	AssertEqual("created", node.Properties["created"], "2025-07-11T14:42:32Z", t)

	// Verify properties defined in the transform config
	AssertEqual("cpuGuest", node.Properties["cpuGuest"], int64(2), t)
	AssertEqual("memoryGuest", node.Properties["memoryGuest"], int64(4294967296), t) // 4Gi
}

func Test_genericResourceFromConfigVirtualMachineClusterInstancetype(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineclusterinstancetype.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "u1.medium", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineClusterInstancetype", t)
	AssertEqual("created", node.Properties["created"], "2025-07-11T14:42:22Z", t)

	// Verify properties defined in the transform config
	AssertEqual("cpuGuest", node.Properties["cpuGuest"], int64(4), t)
	AssertEqual("memoryGuest", node.Properties["memoryGuest"], int64(8589934592), t) // 8Gi
}

func Test_genericResourceFromConfigDataSource(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("datasource.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "DataSourceName", t)
	AssertEqual("kind", node.Properties["kind"], "DataSource", t)
	AssertEqual("created", node.Properties["created"], "2025-07-11T14:41:22Z", t)

	// Verify properties defined in the transform config
	AssertEqual("pvcName", node.Properties["pvcName"], "datasourcePVCName", t)
	AssertEqual("pvcNamespace", node.Properties["pvcNamespace"], "datasourcePVCNamespace", t)
	AssertEqual("snapshotName", node.Properties["snapshotName"], "datasourceSnapshotName", t)
	AssertEqual("snapshotNamespace", node.Properties["snapshotNamespace"], "datasourceSnapshotNamespace", t)
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"ThatType": "True",
		"ThisType": "False",
	}, t)
}

func Test_genericResourceFromConfigVirtualMachineClone(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachineclone.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "full-vm-clone", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineClone", t)
	AssertEqual("created", node.Properties["created"], "2025-07-11T14:42:22Z", t)

	// Verify properties defined in the transform config
	AssertEqual("phase", node.Properties["phase"], "Phased", t)
	AssertEqual("targetName", node.Properties["targetName"], "full-clone-vm", t)
	AssertEqual("targetKind", node.Properties["targetKind"], "RealityMachine", t)
	AssertEqual("sourceName", node.Properties["sourceName"], "source-vm", t)
	AssertEqual("sourceKind", node.Properties["sourceKind"], "VirtualMachine", t)
	AssertDeepEqual("condition", node.Properties["condition"], map[string]string{
		"NotReconciled": "True",
		"Reconciled":    "False",
	}, t)
}

func Test_genericResourceFromConfigNetworkAttachmentDefinition(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("networkattachmentdefinition.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "next-net", t)
	AssertEqual("kind", node.Properties["kind"], "NetworkAttachmentDefinition", t)
	AssertEqual("created", node.Properties["created"], "2000-04-30T16:22:02Z", t)

	// Verify properties defined in the transform config
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"description": "Definition of a network attachment",
		"label":       "test",
	}, t)
}

func Test_genericResourceFromConfigDataImportCron(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("dataimportcron.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "fedora-image-cron", t)
	AssertEqual("kind", node.Properties["kind"], "DataImportCron", t)
	AssertEqual("created", node.Properties["created"], "2025-12-01T14:06:11Z", t)

	// Verify properties defined in the transform config
	AssertEqual("managedDataSource", node.Properties["managedDataSource"], "fedora", t)
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"cdi.kubevirt.io/storage.import.lastCronTime": "2025-12-15T08:04:03Z",
		"cdi.kubevirt.io/storage.import.nextCronTime": "2025-12-15T20:04:00Z",
	}, t)
}

func Test_genericResourceFromConfigVolumeSnapshot(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("volumesnapshot.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "example-volume-snapshot", t)
	AssertEqual("kind", node.Properties["kind"], "VolumeSnapshot", t)
	AssertEqual("created", node.Properties["created"], "2025-12-15T12:00:00Z", t)

	// Verify properties defined in the transform config
	AssertEqual("volumeSnapshotClassName", node.Properties["volumeSnapshotClassName"], "csi-snapshot-class", t)
	AssertEqual("persistentVolumeClaimName", node.Properties["persistentVolumeClaimName"], "example-pvc", t)
	AssertEqual("restoreSize", node.Properties["restoreSize"], int64(10737418240), t) // 10Gi
}

func Test_genericResourceFromConfigMigrationPolicy(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("migrationpolicy.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "example-migration-policy", t)
	AssertEqual("kind", node.Properties["kind"], "MigrationPolicy", t)
	AssertEqual("created", node.Properties["created"], "2025-12-15T12:00:00Z", t)

	// Verify properties defined in the transform config
	AssertEqual("allowAutoConverge", node.Properties["allowAutoConverge"], "true", t)
	AssertEqual("allowPostCopy", node.Properties["allowPostCopy"], "false", t)
	AssertEqual("bandwidthPerMigration", node.Properties["bandwidthPerMigration"], int64(67108864), t)
	AssertEqual("completionTimeoutPerGiB", node.Properties["completionTimeoutPerGiB"], int64(120), t)
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"migrations.kubevirt.io/description": "Migration policy for high-priority workloads",
	}, t)
	AssertDeepEqual("_namespaceSelector", node.Properties["_namespaceSelector"], map[string]string{
		"hpc-workloads": "true", "hpc-nonworkloads": "true",
		"enabled": "true", "priority": "10", "maxInstances": "9223372036854775807", "cpuThreshold": "0.85",
	}, t)
	AssertDeepEqual("_virtualMachineInstanceSelector", node.Properties["_virtualMachineInstanceSelector"], map[string]string{
		"workload-type": "db", "production": "false", "replicas": "3", "memoryLimit": "4294967296", "diskUsage": "75.5",
	}, t)
}

func Test_genericResourceFromConfigConfigMapMatchLabel(t *testing.T) {
	config.Cfg.DeployedInHub = false // temporarily set to false else _hubClusterResource gets appended during full test suite
	defer func() {
		config.Cfg.DeployedInHub = true
	}()
	var r unstructured.Unstructured
	UnmarshalFile("configmap.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "app-config", t)
	AssertEqual("kind", node.Properties["kind"], "ConfigMap", t)
	AssertEqual("created", node.Properties["created"], "2026-01-05T14:27:31Z", t)
	AssertEqual("apiversion", node.Properties["apiversion"], "v1", t)
	AssertEqual("namespace", node.Properties["namespace"], "default", t)
	AssertDeepEqual("label", node.Properties["label"], map[string]string{
		"app": "my-app", "component": "backend", "kiagnose/checkup-type": "true",
	}, t)

	// Verify properties defined in the transform config
	AssertEqual("configParamMaxDesiredLatency", node.Properties["configParamMaxDesiredLatency"], int64(234), t)
	AssertEqual("configParamNADNamespace", node.Properties["configParamNADNamespace"], "NAD-namespace", t)
	AssertEqual("configParamNADName", node.Properties["configParamNADName"], "NAD-name", t)
	AssertEqual("configParamTargetNode", node.Properties["configParamTargetNode"], "spec-param-target-node", t)
	AssertEqual("configParamSourceNode", node.Properties["configParamSourceNode"], "spec-param-source-node", t)
	AssertEqual("configParamSampleDuration", node.Properties["configParamSampleDuration"], int64(123), t)
	AssertEqual("configTimeout", node.Properties["configTimeout"], "10m", t)
	AssertEqual("configCompletionTimestamp", node.Properties["configCompletionTimestamp"], "2027-01-05T14:27:31Z", t)
	AssertEqual("configFailureReason", node.Properties["configFailureReason"], "it broke", t)
	AssertEqual("configStartTimestamp", node.Properties["configStartTimestamp"], "2026-01-05T14:27:31Z", t)
	AssertEqual("configSucceeded", node.Properties["configSucceeded"], "true", t)
	AssertEqual("configStatusAVGLatencyNano", node.Properties["configStatusAVGLatencyNano"], int64(12345), t)
	AssertEqual("configStatusMaxLatencyNano", node.Properties["configStatusMaxLatencyNano"], int64(23456), t)
	AssertEqual("configStatusMinLatencyNano", node.Properties["configStatusMinLatencyNano"], int64(34567), t)
	AssertEqual("configStatusMeasurementDuration", node.Properties["configStatusMeasurementDuration"], int64(123), t)
	AssertEqual("configStatusTargetNode", node.Properties["configStatusTargetNode"], "status-result-target-node", t)
	AssertEqual("configStatusSourceNode", node.Properties["configStatusSourceNode"], "status-result-source-node", t)
	assert.Equal(t, 23, len(node.Properties))
}

func Test_genericResourceFromConfigConfigMapNoMatchLabel(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("configmap.json", &r, t)
	r.SetLabels(map[string]string{"asdf": "true"})
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "app-config", t)
	AssertEqual("kind", node.Properties["kind"], "ConfigMap", t)
	AssertEqual("created", node.Properties["created"], "2026-01-05T14:27:31Z", t)

	// Verify properties defined in the transform config aren't present because they don't match label
	AssertEqual("configParamMaxDesiredLatency", node.Properties["configParamMaxDesiredLatency"], nil, t)
	AssertEqual("configParamNADNamespace", node.Properties["configParamNADNamespace"], nil, t)
	AssertEqual("configParamNADName", node.Properties["configParamNADName"], nil, t)
	AssertEqual("configParamTargetNode", node.Properties["configParamTargetNode"], nil, t)
	AssertEqual("configParamSourceNode", node.Properties["configParamSourceNode"], nil, t)
	AssertEqual("configParamSampleDuration", node.Properties["configParamSampleDuration"], nil, t)
	AssertEqual("configTimeout", node.Properties["configTimeout"], nil, t)
	AssertEqual("configCompletionTimestamp", node.Properties["configCompletionTimestamp"], nil, t)
	AssertEqual("configFailureReason", node.Properties["configFailureReason"], nil, t)
	AssertEqual("configStartTimestamp", node.Properties["configStartTimestamp"], nil, t)
	AssertEqual("configSucceeded", node.Properties["configSucceeded"], nil, t)
	AssertEqual("configStatusAVGLatencyNano", node.Properties["configStatusAVGLatencyNano"], nil, t)
	AssertEqual("configStatusMaxLatencyNano", node.Properties["configStatusMaxLatencyNano"], nil, t)
	AssertEqual("configStatusMinLatencyNano", node.Properties["configStatusMinLatencyNano"], nil, t)
	AssertEqual("configStatusMeasurementDuration", node.Properties["configStatusMeasurementDuration"], nil, t)
	AssertEqual("configStatusTargetNode", node.Properties["configStatusTargetNode"], nil, t)
	AssertEqual("configStatusSourceNode", node.Properties["configStatusSourceNode"], nil, t)
}

func Test_genericResourceFromConfigMapNoLabel(t *testing.T) {
	config.Cfg.DeployedInHub = false // temporarily set to false else _hubClusterResource gets appended during full test suite
	defer func() {
		config.Cfg.DeployedInHub = true
	}()
	var r unstructured.Unstructured
	UnmarshalFile("configmap-two.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "app-config", t)
	AssertEqual("kind", node.Properties["kind"], "ConfigMap", t)
	AssertEqual("created", node.Properties["created"], "2026-01-05T14:27:31Z", t)
	AssertEqual("apiversion", node.Properties["apiversion"], "v1", t)
	AssertEqual("namespace", node.Properties["namespace"], "default", t)
	AssertDeepEqual("label", node.Properties["label"], map[string]string{
		"app": "my-app", "component": "backend",
	}, t)

	// Verify that there's no more indexed properties than the common ones
	assert.Equal(t, 6, len(node.Properties))
}

func Test_genericResourceFromConfigTemplateMatchLabel(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("template.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos-stream9-desktop-large", t)
	AssertEqual("kind", node.Properties["kind"], "Template", t)
	AssertEqual("created", node.Properties["created"], "2026-01-07T22:12:17Z", t)

	// Verify properties defined in the transform config
	AssertEqual("objectVMArchitecture", node.Properties["objectVMArchitecture"], "amd64", t)
	AssertEqual("objectVMName", node.Properties["objectVMName"], "${NAME}", t)
}

func Test_genericResourceFromConfigTemplateNoMatchLabel(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("template.json", &r, t)
	r.SetLabels(map[string]string{"asdf": "true"})
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos-stream9-desktop-large", t)
	AssertEqual("kind", node.Properties["kind"], "Template", t)
	AssertEqual("created", node.Properties["created"], "2026-01-07T22:12:17Z", t)

	// Verify properties defined in the transform config aren't present because they don't match label
	AssertEqual("objectVMArchitecture", node.Properties["objectVMArchitecture"], nil, t)
	AssertEqual("objectVMName", node.Properties["objectVMName"], nil, t)
}

// TestBooleanFieldsStoredAsStrings_GenericConfig verifies that boolean fields
// extracted via genericResourceConfig are stored as their string representations
// so the search API can query them correctly. The API queries JSONB fields using
// the '?' key-exists operator which only matches strings — a raw JSON boolean
// is unreachable via "field:true" queries. See: https://issues.redhat.com/browse/ACM-30764
func TestBooleanFieldsStoredAsStrings_GenericConfig(t *testing.T) {
	tests := []struct {
		name     string
		resource *unstructured.Unstructured
		field    string
		expected string
	}{
		{
			name:     "StorageClass allowVolumeExpansion=true",
			field:    "allowVolumeExpansion",
			expected: "true",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "storage.k8s.io/v1", "kind": "StorageClass",
				"metadata":             map[string]interface{}{"name": "test"},
				"allowVolumeExpansion": true,
			}},
		},
		{
			name:     "StorageClass allowVolumeExpansion=false",
			field:    "allowVolumeExpansion",
			expected: "false",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "storage.k8s.io/v1", "kind": "StorageClass",
				"metadata":             map[string]interface{}{"name": "test"},
				"allowVolumeExpansion": false,
			}},
		},
		{
			name:     "MigrationPolicy allowAutoConverge=true",
			field:    "allowAutoConverge",
			expected: "true",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "migrations.kubevirt.io/v1alpha1", "kind": "MigrationPolicy",
				"metadata": map[string]interface{}{"name": "test"},
				"spec":     map[string]interface{}{"allowAutoConverge": true},
			}},
		},
		{
			name:     "MigrationPolicy allowAutoConverge=false",
			field:    "allowAutoConverge",
			expected: "false",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "migrations.kubevirt.io/v1alpha1", "kind": "MigrationPolicy",
				"metadata": map[string]interface{}{"name": "test"},
				"spec":     map[string]interface{}{"allowAutoConverge": false},
			}},
		},
		{
			name:     "MigrationPolicy allowPostCopy=true",
			field:    "allowPostCopy",
			expected: "true",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "migrations.kubevirt.io/v1alpha1", "kind": "MigrationPolicy",
				"metadata": map[string]interface{}{"name": "test"},
				"spec":     map[string]interface{}{"allowPostCopy": true},
			}},
		},
		{
			name:     "MigrationPolicy allowPostCopy=false",
			field:    "allowPostCopy",
			expected: "false",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "migrations.kubevirt.io/v1alpha1", "kind": "MigrationPolicy",
				"metadata": map[string]interface{}{"name": "test"},
				"spec":     map[string]interface{}{"allowPostCopy": false},
			}},
		},
		{
			name:     "VirtualMachineSnapshot readyToUse=true",
			field:    "readyToUse",
			expected: "true",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "snapshot.kubevirt.io/v1beta1", "kind": "VirtualMachineSnapshot",
				"metadata": map[string]interface{}{"name": "test"},
				"status":   map[string]interface{}{"readyToUse": true},
			}},
		},
		{
			name:     "VirtualMachineSnapshot readyToUse=false",
			field:    "readyToUse",
			expected: "false",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "snapshot.kubevirt.io/v1beta1", "kind": "VirtualMachineSnapshot",
				"metadata": map[string]interface{}{"name": "test"},
				"status":   map[string]interface{}{"readyToUse": false},
			}},
		},
		{
			name:     "VirtualMachineRestore complete=true",
			field:    "complete",
			expected: "true",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "snapshot.kubevirt.io/v1beta1", "kind": "VirtualMachineRestore",
				"metadata": map[string]interface{}{"name": "test"},
				"status":   map[string]interface{}{"complete": true},
			}},
		},
		{
			name:     "VirtualMachineRestore complete=false",
			field:    "complete",
			expected: "false",
			resource: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "snapshot.kubevirt.io/v1beta1", "kind": "VirtualMachineRestore",
				"metadata": map[string]interface{}{"name": "test"},
				"status":   map[string]interface{}{"complete": false},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := GenericResourceBuilder(tc.resource).BuildNode()
			val, ok := node.Properties[tc.field].(string)
			if !ok {
				t.Fatalf("%s should be a string, got %T: %v",
					tc.field, node.Properties[tc.field], node.Properties[tc.field])
			}
			AssertEqual(tc.field, val, tc.expected, t)
		})
	}
}


// ---- ACM-21895: Tests verifying applyDefaultTransformConfig is wired into specific-kind builders ----

// setupTransformConfig sets mergedTransformConfig with a single custom field
// for the given "Kind.apiGroup" key and returns a teardown func.
func setupTransformConfig(t *testing.T, key string, prop ExtractProperty) func() {
	t.Helper()
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		key: {properties: []ExtractProperty{prop}},
	}
	return func() { mergedTransformConfig = orig }
}

// setupConditionsConfig enables extractConditions for the given key.
func setupConditionsConfig(t *testing.T, key string) func() {
	t.Helper()
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		key: {extractConditions: true},
	}
	return func() { mergedTransformConfig = orig }
}

// ---- Deployment -------------------------------------------------------

func TestDeployment_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Deployment.apps", ExtractProperty{
		Name: "strategy", JSONPath: `{.spec.strategy.type}`,
	})()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"spec":       map[string]interface{}{"strategy": map[string]interface{}{"type": "RollingUpdate"}},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "RollingUpdate", node.Properties["strategy"],
		"custom field from CollectorConfig must be extracted by DeploymentResourceBuilder")
}

func TestDeployment_CollectConditions(t *testing.T) {
	defer setupConditionsConfig(t, "Deployment.apps")()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Available", "status": "True"},
			},
		},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.NotNil(t, node.Properties["condition"],
		"collectConditions must produce a condition property in DeploymentResourceBuilder")
}

// ---- DaemonSet --------------------------------------------------------

func TestDaemonSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "DaemonSet.apps", ExtractProperty{
		Name: "updateStrategy", JSONPath: `{.spec.updateStrategy.type}`,
	})()

	d := appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "DaemonSet",
		"spec": map[string]interface{}{
			"updateStrategy": map[string]interface{}{"type": "OnDelete"},
		},
	}}

	node := DaemonSetResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "OnDelete", node.Properties["updateStrategy"],
		"custom field from CollectorConfig must be extracted by DaemonSetResourceBuilder")
}

// ---- StatefulSet -------------------------------------------------------

func TestStatefulSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "StatefulSet.apps", ExtractProperty{
		Name: "serviceName", JSONPath: `{.spec.serviceName}`,
	})()

	s := appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"spec":       map[string]interface{}{"serviceName": "my-service"},
	}}

	node := StatefulSetResourceBuilder(&s, r).BuildNode()
	assert.Equal(t, "my-service", node.Properties["serviceName"],
		"custom field from CollectorConfig must be extracted by StatefulSetResourceBuilder")
}

// ---- ReplicaSet -------------------------------------------------------

func TestReplicaSet_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "ReplicaSet.apps", ExtractProperty{
		Name: "minReadySeconds", JSONPath: `{.spec.minReadySeconds}`,
	})()

	rs := appsv1.ReplicaSet{TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "ReplicaSet",
		"spec":       map[string]interface{}{"minReadySeconds": int64(10)},
	}}

	node := ReplicaSetResourceBuilder(&rs, r).BuildNode()
	assert.Equal(t, int64(10), node.Properties["minReadySeconds"],
		"custom field from CollectorConfig must be extracted by ReplicaSetResourceBuilder")
}

// ---- CronJob ----------------------------------------------------------

func TestCronJob_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "CronJob.batch", ExtractProperty{
		Name: "successfulJobsHistoryLimit", JSONPath: `{.spec.successfulJobsHistoryLimit}`,
	})()

	c := batchv1beta1.CronJob{TypeMeta: metav1.TypeMeta{Kind: "CronJob", APIVersion: "batch/v1beta1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "batch/v1beta1",
		"kind":       "CronJob",
		"spec":       map[string]interface{}{"successfulJobsHistoryLimit": int64(5)},
	}}

	node := CronJobResourceBuilder(&c, r).BuildNode()
	assert.Equal(t, int64(5), node.Properties["successfulJobsHistoryLimit"],
		"custom field from CollectorConfig must be extracted by CronJobResourceBuilder")
}

// ---- PersistentVolume -------------------------------------------------

func TestPersistentVolume_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "PersistentVolume", ExtractProperty{
		Name: "storageClassName", JSONPath: `{.spec.storageClassName}`,
	})()

	p := corev1.PersistentVolume{TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "PersistentVolume",
		"spec":       map[string]interface{}{"storageClassName": "fast"},
	}}

	node := PersistentVolumeResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "fast", node.Properties["storageClassName"],
		"custom field from CollectorConfig must be extracted by PersistentVolumeResourceBuilder")
}

// ---- Subscription (CRD-backed ACM type) --------------------------------

func TestSubscription_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Subscription.apps.open-cluster-management.io", ExtractProperty{
		Name: "placement", JSONPath: `{.spec.placement.local}`,
	})()

	s := acmapp.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Subscription",
			APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Subscription",
		"spec": map[string]interface{}{
			"placement": map[string]interface{}{"local": true},
		},
	}}

	node := SubscriptionResourceBuilder(&s, r).BuildNode()
	// JSONPath extracts booleans as strings
	assert.Equal(t, "true", node.Properties["placement"],
		"custom field from CollectorConfig must be extracted by SubscriptionResourceBuilder")
}

// ---- Wildcard kind: collectConditions on all apps resources -----------

func TestDeployment_WildcardGroupConditions(t *testing.T) {
	// Wildcard key "*.apps" enables conditions for all resources in the apps group
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		"*.apps": {extractConditions: true},
	}
	defer func() { mergedTransformConfig = orig }()

	d := appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Progressing", "status": "True"},
			},
		},
	}}

	node := DeploymentResourceBuilder(&d, r).BuildNode()
	assert.NotNil(t, node.Properties["condition"],
		"wildcard *.apps collectConditions must produce condition property in Deployment")
}

// ---- ConfigPolicyResourceBuilder (unstructured-native) -----------------

func TestConfigPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"ConfigurationPolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "remediationAction", JSONPath: `{.spec.remediationAction}`})()

	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "ConfigurationPolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"remediationAction": "enforce"},
	}}

	node := ConfigPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "enforce", node.Properties["remediationAction"],
		"custom field from CollectorConfig must be extracted by ConfigPolicyResourceBuilder")
}

// ---- OperatorPolicyResourceBuilder (unstructured-native) ---------------

func TestOperatorPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"OperatorPolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "complianceType", JSONPath: `{.spec.complianceType}`})()

	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1beta1",
		"kind":       "OperatorPolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"complianceType": "musthave"},
	}}

	node := OperatorPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "musthave", node.Properties["complianceType"],
		"custom field from CollectorConfig must be extracted by OperatorPolicyResourceBuilder")
}

// ---- KyvernoPolicyResourceBuilder (unstructured-native) ----------------

func TestKyvernoPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"ClusterPolicy.kyverno.io",
		ExtractProperty{Name: "failureAction", JSONPath: `{.spec.validationFailureAction}`})()

	p := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata":   map[string]interface{}{"name": "test"},
		"spec":       map[string]interface{}{"validationFailureAction": "enforce"},
	}}

	node := KyvernoPolicyResourceBuilder(p).BuildNode()
	assert.Equal(t, "enforce", node.Properties["failureAction"],
		"custom field from CollectorConfig must be extracted by KyvernoPolicyResourceBuilder")
}

// ---- Verify specific-kind properties still set correctly (regression) --

func TestDeployment_SpecificPropertiesUnaffected(t *testing.T) {
	var desired int32 = 3
	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 2,
			Replicas:          3,
			ReadyReplicas:     2,
		},
		Spec: appsv1.DeploymentSpec{Replicas: &desired},
	}

	node := DeploymentResourceBuilder(&d, &unstructured.Unstructured{}).BuildNode()
	assert.Equal(t, int64(2), node.Properties["available"])
	assert.Equal(t, int64(3), node.Properties["current"])
	assert.Equal(t, int64(2), node.Properties["ready"])
	assert.Equal(t, int64(3), node.Properties["desired"])
}

// ---- DeploymentConfig (OpenShift) -------------------------------------

func TestDeploymentConfig_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "DeploymentConfig.apps.openshift.io", ExtractProperty{
		Name: "strategy", JSONPath: `{.spec.strategy.type}`,
	})()

	d := ocpapp.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{Kind: "DeploymentConfig", APIVersion: "apps.openshift.io/v1"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.openshift.io/v1",
		"kind":       "DeploymentConfig",
		"spec":       map[string]interface{}{"strategy": map[string]interface{}{"type": "Recreate"}},
	}}

	node := DeploymentConfigResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "Recreate", node.Properties["strategy"],
		"custom field must be extracted by DeploymentConfigResourceBuilder")
}

// ---- ArgoApplication --------------------------------------------------

func TestArgoApplication_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Application.argoproj.io", ExtractProperty{
		Name: "project", JSONPath: `{.spec.project}`,
	})()

	a := ArgoApplication{
		TypeMeta: metav1.TypeMeta{Kind: "Application", APIVersion: "argoproj.io/v1alpha1"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"spec":       map[string]interface{}{"project": "default"},
	}}

	node := ArgoApplicationResourceBuilder(&a, r).BuildNode()
	assert.Equal(t, "default", node.Properties["project"],
		"custom field must be extracted by ArgoApplicationResourceBuilder")
}

// ---- Channel ----------------------------------------------------------

func TestChannel_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Channel.apps.open-cluster-management.io", ExtractProperty{
		Name: "secretRef", JSONPath: `{.spec.secretRef.name}`,
	})()

	c := acmchannel.Channel{
		TypeMeta: metav1.TypeMeta{
			Kind: "Channel", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Channel",
		"spec":       map[string]interface{}{"secretRef": map[string]interface{}{"name": "my-secret"}},
	}}

	node := ChannelResourceBuilder(&c, r).BuildNode()
	assert.Equal(t, "my-secret", node.Properties["secretRef"],
		"custom field must be extracted by ChannelResourceBuilder")
}

// ---- AppDeployable ----------------------------------------------------

func TestAppDeployable_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "Deployable.apps.open-cluster-management.io", ExtractProperty{
		Name: "deployer", JSONPath: `{.spec.deployer.kind}`,
	})()

	d := appDeployable.Deployable{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployable", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Deployable",
		"spec":       map[string]interface{}{"deployer": map[string]interface{}{"kind": "Helm"}},
	}}

	node := AppDeployableResourceBuilder(&d, r).BuildNode()
	assert.Equal(t, "Helm", node.Properties["deployer"],
		"custom field must be extracted by AppDeployableResourceBuilder")
}

// ---- AppHelmCR --------------------------------------------------------

func TestAppHelmCR_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "HelmRelease.apps.open-cluster-management.io", ExtractProperty{
		Name: "version", JSONPath: `{.spec.version}`,
	})()

	a := acmhelmrelease.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind: "HelmRelease", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "HelmRelease",
		"spec":       map[string]interface{}{"version": "1.2.3"},
	}}

	node := AppHelmCRResourceBuilder(&a, r).BuildNode()
	assert.Equal(t, "1.2.3", node.Properties["version"],
		"custom field must be extracted by AppHelmCRResourceBuilder")
}

// ---- KlusterletAddonConfig --------------------------------------------

func TestKlusterletAddonConfig_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"KlusterletAddonConfig.agent.open-cluster-management.io",
		ExtractProperty{Name: "clusterName", JSONPath: `{.spec.clusterName}`})()

	p := agentv1.KlusterletAddonConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "KlusterletAddonConfig", APIVersion: "agent.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "agent.open-cluster-management.io/v1",
		"kind":       "KlusterletAddonConfig",
		"spec":       map[string]interface{}{"clusterName": "my-cluster"},
	}}

	node := KlusterletAddonConfigResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "my-cluster", node.Properties["clusterName"],
		"custom field must be extracted by KlusterletAddonConfigResourceBuilder")
}

// ---- PlacementBinding -------------------------------------------------

func TestPlacementBinding_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"PlacementBinding.apps.open-cluster-management.io",
		ExtractProperty{Name: "bindingOverrides", JSONPath: `{.bindingOverrides.remediationAction}`})()

	p := gvrpolicy.PlacementBinding{
		TypeMeta: metav1.TypeMeta{
			Kind: "PlacementBinding", APIVersion: "apps.open-cluster-management.io/v1",
		},
		PlacementRef: gvrpolicy.PlacementSubject{Name: "my-placement", Kind: "PlacementRule"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion":       "apps.open-cluster-management.io/v1",
		"kind":             "PlacementBinding",
		"bindingOverrides": map[string]interface{}{"remediationAction": "enforce"},
	}}

	node := PlacementBindingResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "enforce", node.Properties["bindingOverrides"],
		"custom field must be extracted by PlacementBindingResourceBuilder")
}

// ---- PlacementRule ----------------------------------------------------

func TestPlacementRule_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"PlacementRule.apps.open-cluster-management.io",
		ExtractProperty{Name: "clusterConditions", JSONPath: `{.spec.clusterConditions[0].type}`})()

	p := acmrule.PlacementRule{
		TypeMeta: metav1.TypeMeta{
			Kind: "PlacementRule", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "PlacementRule",
		"spec": map[string]interface{}{
			"clusterConditions": []interface{}{
				map[string]interface{}{"type": "ManagedClusterConditionAvailable"},
			},
		},
	}}

	node := PlacementRuleResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "ManagedClusterConditionAvailable", node.Properties["clusterConditions"],
		"custom field must be extracted by PlacementRuleResourceBuilder")
}

// ---- PolicyResourceBuilder (typed) ------------------------------------

func TestPolicy_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t,
		"Policy.policy.open-cluster-management.io",
		ExtractProperty{Name: "severity", JSONPath: `{.metadata.annotations.policy\.open-cluster-management\.io/severity}`})()

	p := gvrpolicy.Policy{
		TypeMeta: metav1.TypeMeta{
			Kind: "Policy", APIVersion: "policy.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "Policy",
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"policy.open-cluster-management.io/severity": "high",
			},
		},
	}}

	node := PolicyResourceBuilder(&p, r).BuildNode()
	assert.Equal(t, "high", node.Properties["severity"],
		"custom field must be extracted by PolicyResourceBuilder")
}

// ---- CertPolicyResourceBuilder (tests the early-return path) ----------

func TestCertPolicy_CustomFieldFromCollectorConfig_EarlyReturn(t *testing.T) {
	defer setupTransformConfig(t,
		"CertificatePolicy.policy.open-cluster-management.io",
		ExtractProperty{Name: "minimumDuration", JSONPath: `{.spec.minimumDuration}`})()

	// Empty status causes the early return path in CertPolicyResourceBuilder
	c := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "policy.open-cluster-management.io/v1",
		"kind":       "CertificatePolicy",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec":       map[string]interface{}{"minimumDuration": "300h"},
	}}

	node := CertPolicyResourceBuilder(c).BuildNode()
	assert.Equal(t, "300h", node.Properties["minimumDuration"],
		"custom field must be extracted even via the early-return path of CertPolicyResourceBuilder")
}

// ---- PolicyReportResourceBuilder --------------------------------------

func TestPolicyReport_CustomFieldFromCollectorConfig(t *testing.T) {
	defer setupTransformConfig(t, "PolicyReport.wgpolicyk8s.io", ExtractProperty{
		Name: "reportType", JSONPath: `{.metadata.labels.app\.kubernetes\.io/name}`,
	})()

	pr := PolicyReport{
		TypeMeta: metav1.TypeMeta{Kind: "PolicyReport", APIVersion: "wgpolicyk8s.io/v1alpha2"},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "wgpolicyk8s.io/v1alpha2",
		"kind":       "PolicyReport",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"app.kubernetes.io/name": "kyverno",
			},
		},
	}}

	node := PolicyReportResourceBuilder(&pr, r).BuildNode()
	assert.Equal(t, "kyverno", node.Properties["reportType"],
		"custom field must be extracted by PolicyReportResourceBuilder")
}

// ---- additionalColumns pass-through (CRD-backed kind) -----------------

func TestSubscription_AdditionalColumnsPassThrough(t *testing.T) {
	// Verify that CRD additionalPrinterColumns from the informer flow through
	// to the node when a CollectorConfig sets additionalPrinterColumnsPriority.
	priority := 0
	orig := mergedTransformConfig
	mergedTransformConfig = map[string]ResourceConfig{
		"Subscription.apps.open-cluster-management.io": {
			additionalPrinterColumnsPriority: &priority,
		},
	}
	defer func() { mergedTransformConfig = orig }()

	s := acmapp.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind: "Subscription", APIVersion: "apps.open-cluster-management.io/v1",
		},
	}
	r := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.open-cluster-management.io/v1",
		"kind":       "Subscription",
	}}
	additionalCol := ExtractProperty{Name: "phase", JSONPath: `{.status.phase}`}
	r.Object["status"] = map[string]interface{}{"phase": "Subscribed"}

	node := SubscriptionResourceBuilder(&s, r, additionalCol).BuildNode()
	assert.Equal(t, "Subscribed", node.Properties["phase"],
		"additionalColumns must flow through SubscriptionResourceBuilder variadic param")
}
