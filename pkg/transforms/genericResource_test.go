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
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("osName", node.Properties["osName"], "rhel9", t)
	AssertDeepEqual("pvcClaimNames", node.Properties["pvcClaimNames"],
		[]interface{}{"the-claim-is-persistent", "the-claim-is-too-persistent"}, t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("runStrategy", node.Properties["runStrategy"], "always", t)
	AssertEqual("status", node.Properties["status"], "Running", t)
	AssertEqual("workload", node.Properties["workload"], "server", t)
	AssertEqual("_specRunning", node.Properties["_specRunning"], true, t)
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
	AssertDeepEqual("affinity", node.Properties["affinity"], map[string]interface{}{
		"nodeAffinity": map[string]interface{}{
			"requiredDuringSchedulingIgnoredDuringExecution": map[string]interface{}{
				"nodeSelectorTerms": []interface{}{
					map[string]interface{}{
						"matchExpressions": []interface{}{
							map[string]interface{}{
								"key":      "node-role.kubernetes.io/worker",
								"operator": "Exists",
							},
						},
					},
				},
			},
		},
		"podAntiAffinity": map[string]interface{}{
			"preferredDuringSchedulingIgnoredDuringExecution": []interface{}{
				map[string]interface{}{
					"PodAffinityTerm": map[string]interface{}{
						"labelSelector": map[string]interface{}{
							"matchExpressions": []interface{}{
								map[string]interface{}{
									"key":      "kubevirt.io/domain",
									"operator": "In",
									"values":   []interface{}{"my-ha-vm"},
								},
							},
						},
						"topologyKey": "kubernetes.io/hostname",
					},
				},
			},
		},
	}, t)
	AssertEqual("cpu", node.Properties["cpu"], int64(1), t)
	AssertEqual("cpuSockets", node.Properties["cpuSockets"], int64(1), t)
	AssertEqual("cpuThreads", node.Properties["cpuThreads"], int64(1), t)
	AssertDeepEqual("gpuNames", node.Properties["gpuNames"], []interface{}{"gpu-one", "gpu-two"}, t)
	AssertEqual("guestOSInfoID", node.Properties["guestOSInfoID"], "centos", t)
	AssertDeepEqual("hostDeviceNames", node.Properties["hostDeviceNames"], []interface{}{"host-device-one", "host-device-two"}, t)
	AssertDeepEqual("interfaceNames", node.Properties["interfaceNames"], []interface{}{"default", "non-default"}, t)
	AssertDeepEqual("interfaceStatusInterfaceNames", node.Properties["interfaceStatusInterfaceNames"], []interface{}{"eth0", "eth0-2"}, t)
	AssertDeepEqual("interfaceStatusIPAddresses", node.Properties["interfaceStatusIPAddresses"], []interface{}{"10.128.1.193", "10.128.1.194"}, t)
	AssertDeepEqual("interfaceStatusNames", node.Properties["interfaceStatusNames"], []interface{}{"default", "default2"}, t)
	AssertEqual("ipaddress", node.Properties["ipaddress"], "10.128.1.193", t)
	AssertEqual("liveMigratable", node.Properties["liveMigratable"], "False", t)
	AssertEqual("memory", node.Properties["memory"], int64(2147483648), t) // 2Gi
	AssertEqual("migrationPolicyName", node.Properties["migrationPolicyName"], "my-migration-policy", t)
	AssertDeepEqual("multusNetworkNames", node.Properties["multusNetworkNames"], []interface{}{"multus-one", "multus-two"}, t)
	AssertDeepEqual("networkNames", node.Properties["networkNames"], []interface{}{"default", "non-default"}, t)
	AssertEqual("node", node.Properties["node"], "sno-0-0", t)
	AssertEqual("osVersion", node.Properties["osVersion"], "7 (Core)", t)
	AssertEqual("phase", node.Properties["phase"], "Running", t)
	AssertEqual("ready", node.Properties["ready"], "True", t)
	AssertEqual("startStrategy", node.Properties["startStrategy"], "Paused", t)
	AssertDeepEqual("tolerations", node.Properties["tolerations"], []interface{}{
		map[string]interface{}{"effect": "NoSchedule", "key": "node-role.kubernetes.io/infra", "operator": "Exists"},
		map[string]interface{}{"effect": "NoExecute", "key": "dedicated", "operator": "Equal"},
	}, t)
	AssertEqual("vmSize", node.Properties["vmSize"], "small", t)
	AssertDeepEqual("volumes", node.Properties["volumes"], []interface{}{
		map[string]interface{}{"dataVolume": map[string]interface{}{"name": "centos7-gray-owl-35"}, "name": "rootdisk"},
		map[string]interface{}{"emptyDisk": map[string]interface{}{"capacity": "2Gi"}, "name": "emptydisk"},
	}, t)
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
	AssertEqual("readyToUse", node.Properties["readyToUse"], true, t)
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
	AssertEqual("complete", node.Properties["complete"], true, t)
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
	AssertEqual("allowVolumeExpansion", node.Properties["allowVolumeExpansion"], true, t)
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
	AssertEqual("config", node.Properties["config"],
		"{\n  \"cniVersion\": \"0.3.1\",\n  \"name\": \"work-network\",\n  \"namespace\": \"namespace2\","+
			"\n  \"type\": \"host-device\",\n  \"device\": \"eth1\",\n  \"ipam\": {\n    \"type\": \"dhcp\"\n  }\n}", t)
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
	AssertEqual("allowAutoConverge", node.Properties["allowAutoConverge"], true, t)
	AssertEqual("allowPostCopy", node.Properties["allowPostCopy"], false, t)
	AssertEqual("bandwidthPerMigration", node.Properties["bandwidthPerMigration"], int64(67108864), t)
	AssertEqual("completionTimeoutPerGiB", node.Properties["completionTimeoutPerGiB"], int64(120), t)
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"migrations.kubevirt.io/description": "Migration policy for high-priority workloads",
	}, t)
	AssertDeepEqual("selectors", node.Properties["selectors"], map[string]interface{}{
		"namespaceSelector": map[string]interface{}{
			"matchNames": []interface{}{"default", "production"},
		},
		"virtualMachineInstanceSelector": map[string]interface{}{
			"matchLabels": map[string]interface{}{"workload": "critical"},
		},
	}, t)
}

func Test_genericResourceFromConfigVirtualMachineSnapshotContent(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("virtualmachinesnapshotcontent.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "vmsnapshotcontent-asdf", t)
	AssertEqual("kind", node.Properties["kind"], "VirtualMachineSnapshotContent", t)
	AssertEqual("created", node.Properties["created"], "2025-01-05T14:12:33Z", t)
}

func Test_genericResourceFromConfigConfigMap(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("configmap.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "app-config", t)
	AssertEqual("kind", node.Properties["kind"], "ConfigMap", t)
	AssertEqual("created", node.Properties["created"], "2026-01-05T14:27:31Z", t)

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
}

func Test_genericResourceFromConfigTemplate(t *testing.T) {
	var r unstructured.Unstructured
	UnmarshalFile("template.json", &r, t)
	node := GenericResourceBuilder(&r).BuildNode()

	// Verify common properties
	AssertEqual("name", node.Properties["name"], "centos-stream9-desktop-large", t)
	AssertEqual("kind", node.Properties["kind"], "Template", t)
	AssertEqual("created", node.Properties["created"], "2026-01-07T22:12:17Z", t)

	// Verify properties defined in the transform config
	AssertDeepEqual("annotation", node.Properties["annotation"], map[string]string{
		"template.kubevirt.io/provider":               "Red Hat",
		"name.os.template.kubevirt.io/centos-stream9": "CentOS Stream 9 or higher",
		"template.kubevirt.io/provider-url":           "https://www.centos.org",
		"template.kubevirt.io/containerdisks":         "quay.io/containerdisks/centos-stream:9\n",
		"template.kubevirt.io/version":                "v1alpha1",
	}, t)
	AssertDeepEqual("objectAnnotations", node.Properties["objectAnnotations"], map[string]interface{}{
		"vm.kubevirt.io/validations": "[{\"name\":\"minimal-required-memory\",\"path\":\"jsonpath::.spec.domain.memory.guest\",\"rule\":\"integer\",\"message\":\"This VM requires more memory.\",\"min\":1610612736}]",
	}, t)
	AssertDeepEqual("objectDataVolumeTemplates", node.Properties["objectDataVolumeTemplates"], []interface{}{
		map[string]interface{}{
			"apiVersion": "cdi.kubevirt.io/v1beta1",
			"kind":       "DataVolume",
			"metadata": map[string]interface{}{
				"name": "${NAME}",
			},
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"kind":      "DataSource",
					"name":      "${DATA_SOURCE_NAME}",
					"namespace": "${DATA_SOURCE_NAMESPACE}",
				},
				"storage": map[string]interface{}{
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"storage": "30Gi",
						},
					},
				},
			},
		},
	}, t)
	AssertDeepEqual("objectLabels", node.Properties["objectLabels"], map[string]interface{}{
		"app": "${NAME}",
		"kubevirt.io/dynamic-credentials-support": "true",
		"vm.kubevirt.io/template":                 "centos-stream9-desktop-large",
		"vm.kubevirt.io/template.revision":        "1",
		"vm.kubevirt.io/template.version":         "v0.34.1",
	}, t)
	AssertEqual("objectVMArchitecture", node.Properties["objectVMArchitecture"], "amd64", t)
	AssertEqual("objectVMName", node.Properties["objectVMName"], "${NAME}", t)
	AssertDeepEqual("objectParameters", node.Properties["objectParameters"], []interface{}{
		map[string]interface{}{
			"description": "VM name",
			"from":        "centos-stream9-[a-z0-9]{16}",
			"generate":    "expression",
			"name":        "NAME",
		},
		map[string]interface{}{
			"description": "Name of the DataSource to clone",
			"name":        "DATA_SOURCE_NAME",
			"value":       "centos-stream9",
		},
		map[string]interface{}{
			"description": "Namespace of the DataSource",
			"name":        "DATA_SOURCE_NAMESPACE",
			"value":       "namespace-1",
		},
		map[string]interface{}{
			"description": "Randomized password for the cloud-init user centos",
			"name":        "CLOUD_USER_PASSWORD",
			"value":       "asdf",
		},
	}, t)
	AssertDeepEqual("objectVolumes", node.Properties["objectVolumes"], []interface{}{
		map[string]interface{}{
			"dataVolume": map[string]interface{}{
				"name": "${NAME}",
			},
			"name": "rootdisk",
		},
		map[string]interface{}{
			"cloudInitNoCloud": map[string]interface{}{
				"userData": "#cloud-config\nuser: user\npassword: ${CLOUD_USER_PASSWORD}\nchpasswd: { expire: False }\n",
			},
			"name": "cloudinitdisk",
		},
	}, t)
}
