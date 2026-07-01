package transforms

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	Name     string   // `json:"name,omitempty"`
	JSONPath string   // `json:"jsonpath,omitempty"`
	DataType DataType // `json:"dataType,omitempty"`
	// Denotes the priority if property is an additionalPrinterColumn
	Priority *int // `json:"priority,omitempty"`
	// matchLabel limits extraction to resources with this label.
	// A property to denote that we should only extract this property if this label matches the resource FUTURE: generalize with matchExpression
	matchLabel string // `json:"matchLabel,omitempty"`
	// An internal property to denote this property should be set on the node's metadata instead.
	metadataOnly bool
}

// ExtractEdge declares an edge relationship to extract from a resource.
type ExtractEdge struct {
	Name   string   // `json:"name,omitempty"`
	ToKind string   // `json:"toKind,omitempty"`
	Type   EdgeType // `json:"type,omitempty"`
}

// DataType indicates the type coercion to apply when extracting a property value.
type DataType string

const (
	DataTypeBytes     DataType = "bytes"
	DataTypeSlice     DataType = "slice"
	DataTypeString    DataType = "string"
	DataTypeNumber    DataType = "number"
	DataTypeMapString DataType = "mapString"
)

// matchLabelKiagnose is the label used to identify kiagnose network-latency checkup ConfigMaps.
const matchLabelKiagnose = "kiagnose/checkup-type"

func stringToDataType(s string) DataType {
	switch s {
	case "bytes":
		return DataTypeBytes
	case "slice":
		return DataTypeSlice
	case "string":
		return DataTypeString
	case "number":
		return DataTypeNumber
	case "mapString":
		return DataTypeMapString
	default:
		return DataTypeString
	}
}

// Declares the properties to extract from a given resource.
type ResourceConfig struct {
	properties                       []ExtractProperty // `json:"properties,omitempty"`
	edges                            []ExtractEdge     // `json:"edges,omitempty"`
	extractAnnotations               bool              // `json:"extractAnnotations,omitempty"`
	extractConditions                bool              // `json:"extractConditions,omitempty"`
	additionalPrinterColumnsPriority *int              // `json:"collectAdditionalPrinterColumnsPriority,omitempty"`
}

var (
	defaultTransformIgnoredFields = map[string]bool{
		// Skip age since this likely duplicates "created" that is set by genericProperties.
		"age": true,
	}
	knownStringArrays = map[string]bool{
		"accessMode": true,
		"category":   true,
		"container":  true,
		"image":      true,
		"port":       true,
		"role":       true,
		"rules":      true,
		"subject":    true,
	}
)

// defaultTransformConfig declares the properties to extract from each resource kind.
// It is extended at runtime by CollectorConfig CRs (collectorconfigs.search.open-cluster-management.io).
var defaultTransformConfig = map[string]ResourceConfig{
	"ClusterServiceVersion.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: ".spec.version"},
			{Name: "display", JSONPath: ".spec.displayName"},
			{Name: "phase", JSONPath: ".status.phase"},
		},
	},
	"ClusterOperator.config.openshift.io": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: `.status.versions[?(@.name=="operator")].version`},
			{Name: "available", JSONPath: `.status.conditions[?(@.type=="Available")].status`},
			{Name: "progressing", JSONPath: `.status.conditions[?(@.type=="Progressing")].status`},
			{Name: "degraded", JSONPath: `.status.conditions[?(@.type=="Degraded")].status`},
		},
	},
	"ConfigMap": {
		properties: []ExtractProperty{
			{
				Name:       "configParamMaxDesiredLatency",
				JSONPath:   `.data.spec\.param\.maxDesiredLatencyMilliseconds`,
				matchLabel: matchLabelKiagnose,
			},
			{
				Name:       "configParamNADNamespace",
				JSONPath:   `.data.spec\.param\.networkAttachmentDefinitionNamespace`,
				matchLabel: matchLabelKiagnose,
			},
			{
				Name:       "configParamNADName",
				JSONPath:   `.data.spec\.param\.networkAttachmentDefinitionName`,
				matchLabel: matchLabelKiagnose,
			},
			{Name: "configParamTargetNode", JSONPath: `.data.spec\.param\.targetNode`, matchLabel: matchLabelKiagnose},
			{Name: "configParamSourceNode", JSONPath: `.data.spec\.param\.sourceNode`, matchLabel: matchLabelKiagnose},
			{
				Name:       "configParamSampleDuration",
				JSONPath:   `.data.spec\.param\.sampleDurationSeconds`,
				matchLabel: matchLabelKiagnose,
			},
			{Name: "configTimeout", JSONPath: `.data.spec\.timeout`, matchLabel: matchLabelKiagnose},
			{
				Name:       "configCompletionTimestamp",
				JSONPath:   `.data.status\.completionTimestamp`,
				matchLabel: matchLabelKiagnose,
			},
			{Name: "configFailureReason", JSONPath: `.data.status\.failureReason`, matchLabel: matchLabelKiagnose},
			{Name: "configStartTimestamp", JSONPath: `.data.status\.startTimestamp`, matchLabel: matchLabelKiagnose},
			{Name: "configSucceeded", JSONPath: `.data.status\.succeeded`, matchLabel: matchLabelKiagnose},
			{
				Name:       "configStatusAVGLatencyNano",
				JSONPath:   `.data.status\.result\.avgLatencyNanoSec`,
				matchLabel: matchLabelKiagnose,
			},
			{
				Name:       "configStatusMaxLatencyNano",
				JSONPath:   `.data.status\.result\.maxLatencyNanoSec`,
				matchLabel: matchLabelKiagnose,
			},
			{
				Name:       "configStatusMinLatencyNano",
				JSONPath:   `.data.status\.result\.minLatencyNanoSec`,
				matchLabel: matchLabelKiagnose,
			},
			{
				Name:       "configStatusMeasurementDuration",
				JSONPath:   `.data.status\.result\.measurementDurationSec`,
				matchLabel: matchLabelKiagnose,
			},
			{Name: "configStatusTargetNode", JSONPath: `.data.status\.result\.targetNode`, matchLabel: matchLabelKiagnose},
			{Name: "configStatusSourceNode", JSONPath: `.data.status\.result\.sourceNode`, matchLabel: matchLabelKiagnose},
		},
	},
	"DataSource.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "pvcName", JSONPath: `.spec.source.pvc.name`},
			{Name: "pvcNamespace", JSONPath: `.spec.source.pvc.namespace`},
			{Name: "snapshotName", JSONPath: `.spec.source.snapshot.name`},
			{Name: "snapshotNamespace", JSONPath: `.spec.source.snapshot.namespace`},
		},
		extractConditions: true,
	},
	"DataVolume.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "size", JSONPath: `.spec.storage.resources.requests.storage`},
			{Name: "snapshotName", JSONPath: `.spec.source.snapshot.name`},
			{Name: "snapshotNamespace", JSONPath: `.spec.source.snapshot.namespace`},
			{Name: "phase", JSONPath: `.status.phase`},
			{Name: "pvcName", JSONPath: `.spec.source.pvc.name`},
			{Name: "pvcNamespace", JSONPath: `.spec.source.pvc.namespace`},
			{Name: "storageClassName", JSONPath: `.spec.storage.storageClassName`},
		},
		extractAnnotations: true,
	},
	"DataImportCron.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "managedDataSource", JSONPath: `.spec.managedDataSource`},
		},
		extractAnnotations: true,
	},
	"Job": {
		properties: []ExtractProperty{
			{Name: "active", JSONPath: `.status.active`},
			{Name: "failed", JSONPath: `.status.failed`},
		},
	},
	"MigrationPolicy.migrations.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "allowAutoConverge", JSONPath: `.spec.allowAutoConverge`},
			{Name: "allowPostCopy", JSONPath: `.spec.allowPostCopy`},
			{Name: "bandwidthPerMigration", JSONPath: `.spec.bandwidthPerMigration`, DataType: DataTypeBytes},
			{Name: "completionTimeoutPerGiB", JSONPath: `.spec.completionTimeoutPerGiB`},
			{Name: "_namespaceSelector", JSONPath: `.spec.selectors.namespaceSelector`, DataType: DataTypeMapString},
			{
				Name:     "_virtualMachineInstanceSelector",
				JSONPath: `.spec.selectors.virtualMachineInstanceSelector`,
				DataType: DataTypeMapString,
			},
		},
		extractAnnotations: true,
	},
	"MultiClusterHub.operator.open-cluster-management.io": {
		extractConditions: true,
	},
	"Namespace": {
		properties: []ExtractProperty{
			{Name: "status", JSONPath: `.status.phase`},
		},
	},
	"NetworkAddonsConfig.networkaddonsoperator.network.kubevirt.io": {
		extractConditions: true,
	},
	"NetworkAttachmentDefinition.k8s.cni.cncf.io": {
		extractAnnotations: true,
	},
	"Node": {
		properties: []ExtractProperty{
			{Name: "ipAddress", JSONPath: `.status.addresses[?(@.type=="InternalIP")].address`},
			{Name: "memoryAllocatable", JSONPath: `.status.allocatable.memory`, DataType: DataTypeBytes},
			{Name: "memoryCapacity", JSONPath: `.status.capacity.memory`, DataType: DataTypeBytes},
		},
		extractConditions: true,
	},
	"PersistentVolumeClaim": {
		properties: []ExtractProperty{
			{Name: "requestedStorage", JSONPath: `.spec.resources.requests.storage`, DataType: DataTypeBytes},
			{Name: "volumeMode", JSONPath: `.spec.volumeMode`},
		},
	},
	"Pod": {
		extractConditions: true,
	},
	"Search.search.open-cluster-management.io": {
		extractConditions: true,
	},
	"StorageClass.storage.k8s.io": {
		properties: []ExtractProperty{
			{Name: "allowVolumeExpansion", JSONPath: `.allowVolumeExpansion`},
			{Name: "provisioner", JSONPath: `.provisioner`},
			{Name: "reclaimPolicy", JSONPath: `.reclaimPolicy`},
			{Name: "volumeBindingMode", JSONPath: `.volumeBindingMode`},
		},
	},
	"Subscription.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "source", JSONPath: ".spec.source"},
			{Name: "package", JSONPath: ".spec.name"},
			{Name: "channel", JSONPath: ".spec.channel"},
			{Name: "installplan", JSONPath: ".status.installedCSV"},
			{Name: "phase", JSONPath: ".status.state"},
		},
	},
	"Template.template.openshift.io": {
		properties: []ExtractProperty{
			{Name: "objectVMName", JSONPath: `.objects[0].metadata.name`, matchLabel: "template.kubevirt.io/type"},
			{
				Name:       "objectVMArchitecture",
				JSONPath:   `.objects[0].spec.template.spec.architecture`,
				matchLabel: "template.kubevirt.io/type",
			},
		},
	},
	"ValidatingAdmissionPolicy.admissionregistration.k8s.io": {
		properties: []ExtractProperty{
			{Name: "paramKind_kind", JSONPath: `.spec.paramKind.kind`, metadataOnly: true},
			{Name: "paramKind_apiVersion", JSONPath: `.spec.paramKind.apiVersion`, metadataOnly: true},
		},
	},
	"VirtualMachine.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "architecture", JSONPath: `.spec.template.spec.architecture`},
			{Name: "agentConnected", JSONPath: `.status.conditions[?(@.type=="AgentConnected")].status`},
			{Name: "cpu", JSONPath: `.spec.template.spec.domain.cpu.cores`},
			{Name: "dataVolumeNames", JSONPath: `.spec.template.spec.volumes[*].dataVolume.name`, DataType: DataTypeSlice},
			{Name: "_description", JSONPath: `.metadata.annotations.description`},
			{Name: "flavor", JSONPath: `.spec.template.metadata.annotations.\vm\.kubevirt\.io/flavor`},
			{Name: "gpuName", JSONPath: `.spec.template.spec.domain.devices.gpus[*].name`, DataType: DataTypeSlice},
			{
				Name:     "hostDeviceName",
				JSONPath: `.spec.template.spec.domain.devices.hostDevices[*].name`,
				DataType: DataTypeSlice,
			},
			{Name: "instancetype", JSONPath: `.spec.instancetype.name`},
			{Name: "memory", JSONPath: `.spec.template.spec.domain.memory.guest`, DataType: DataTypeBytes},
			{Name: "osName", JSONPath: `.spec.template.metadata.annotations.\vm\.kubevirt\.io/os`},
			{Name: "preference", JSONPath: `.spec.preference.name`},
			{
				Name:     "pvcClaimNames",
				JSONPath: `.spec.template.spec.volumes[*].persistentVolumeClaim.claimName`,
				DataType: DataTypeSlice,
			},
			{Name: "ready", JSONPath: `.status.conditions[?(@.type=='Ready')].status`},
			{Name: "runStrategy", JSONPath: `.spec.runStrategy`},
			{Name: "status", JSONPath: `.status.printableStatus`},
			{Name: "_specRunning", JSONPath: `.spec.running`},
			{Name: "_specRunStrategy", JSONPath: `.spec.runStrategy`},
			{Name: "workload", JSONPath: `.spec.template.metadata.annotations.\vm\.kubevirt\.io/workload`},
		},
		edges: []ExtractEdge{
			{Name: "dataVolumeNames", ToKind: "DataVolume", Type: attachedTo},
			{Name: "pvcClaimNames", ToKind: "PersistentVolumeClaim", Type: attachedTo},
		},
		extractConditions: true,
	},
	"VirtualMachineClone.clone.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "targetName", JSONPath: `.spec.target.name`},
			{Name: "targetKind", JSONPath: `.spec.target.kind`},
			{Name: "sourceName", JSONPath: `.spec.source.name`},
			{Name: "sourceKind", JSONPath: `.spec.source.kind`},
			{Name: "phase", JSONPath: `.status.phase`},
		},
		extractConditions: true,
	},
	"VirtualMachineClusterInstancetype.instancetype.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpuGuest", JSONPath: `.spec.cpu.guest`},
			{Name: "memoryGuest", JSONPath: `.spec.memory.guest`, DataType: DataTypeBytes},
		},
	},
	"VirtualMachineInstance.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpu", JSONPath: `.spec.domain.cpu.cores`},
			{Name: "cpuSockets", JSONPath: `.spec.domain.cpu.sockets`},
			{Name: "cpuThreads", JSONPath: `.spec.domain.cpu.threads`},
			{Name: "guestOSInfoID", JSONPath: `.status.guestOSInfo.id`},
			{Name: "interfaceName", JSONPath: `.spec.domain.devices.interfaces[*].name`, DataType: DataTypeSlice},
			{Name: "_interface", JSONPath: `.status.interfaces[*]`},
			{Name: "ipaddress", JSONPath: `.status.interfaces[0].ipAddress`},
			{Name: "liveMigratable", JSONPath: `.status.conditions[?(@.type=='LiveMigratable')].status`},
			{Name: "memory", JSONPath: `.spec.domain.memory.guest`, DataType: DataTypeBytes},
			{Name: "migrationPolicyName", JSONPath: `.status.migrationState.migrationPolicyName`},
			{Name: "node", JSONPath: `.status.nodeName`},
			{Name: "osVersion", JSONPath: `.status.guestOSInfo.version`},
			{Name: "phase", JSONPath: `.status.phase`},
			{Name: "ready", JSONPath: `.status.conditions[?(@.type=='Ready')].status`},
			{Name: "startStrategy", JSONPath: `.spec.startStrategy`},
			{Name: "vmSize", JSONPath: `.metadata.labels.\kubevirt\.io/size`},
		},
	},
	"VirtualMachineInstanceMigration.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "deleted", JSONPath: `.metadata.deletionTimestamp`},
			{Name: "endTime", JSONPath: `.status.migrationState.endTimestamp`},
			{Name: "migrationPolicyName", JSONPath: `.status.migrationState.migrationPolicyName`},
			{Name: "phase", JSONPath: `.status.phase`},
			{Name: "sourceNode", JSONPath: `.status.migrationState.sourceNode`},
			{Name: "sourcePod", JSONPath: `.status.migrationState.sourcePod`},
			{Name: "targetNode", JSONPath: `.status.migrationState.targetNode`},
			{Name: "vmiName", JSONPath: `.spec.vmiName`},
			{Name: "vmiName", JSONPath: `.spec.vmiName`, metadataOnly: true}, // Used to build the edge
		},
		edges: []ExtractEdge{
			{Name: "vmiName", ToKind: "VirtualMachineInstance", Type: migrationOf},
		},
	},
	"VirtualMachineInstancetype.instancetype.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpuGuest", JSONPath: `.spec.cpu.guest`},
			{Name: "memoryGuest", JSONPath: `.spec.memory.guest`, DataType: DataTypeBytes},
		},
	},
	"VirtualMachineSnapshot.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `.status.conditions[?(@.type=='Ready')].status`},
			{Name: "_conditionReadyReason", JSONPath: `.status.conditions[?(@.type=='Ready')].reason`},
			{Name: "phase", JSONPath: `.status.phase`},
			// indications is an array of strings; items are collected separated by semicolons.
			{Name: "indications", JSONPath: `.status.indications`},
			{Name: "sourceKind", JSONPath: `.spec.source.kind`},
			{Name: "sourceName", JSONPath: `.spec.source.name`},
			{Name: "readyToUse", JSONPath: `.status.readyToUse`},
		},
		extractConditions: true,
	},
	"VirtualMachineRestore.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `.status.conditions[?(@.type=='Ready')].status`},
			{Name: "_conditionReadyReason", JSONPath: `.status.conditions[?(@.type=='Ready')].reason`},
			{Name: "restoreTime", JSONPath: `.status.restoreTime`},
			{Name: "complete", JSONPath: `.status.complete`},
			{Name: "targetApiGroup", JSONPath: `.spec.target.apiGroup`},
			{Name: "targetKind", JSONPath: `.spec.target.kind`},
			{Name: "targetName", JSONPath: `.spec.target.name`},
			{Name: "virtualMachineSnapshotName", JSONPath: `.spec.virtualMachineSnapshotName`},
		},
	},
	"VolumeSnapshot.snapshot.storage.k8s.io": {
		properties: []ExtractProperty{
			{Name: "volumeSnapshotClassName", JSONPath: `.spec.volumeSnapshotClassName`},
			{Name: "persistentVolumeClaimName", JSONPath: `.spec.source.persistentVolumeClaimName`},
			{Name: "restoreSize", JSONPath: `.status.restoreSize`, DataType: DataTypeBytes},
		},
	},
}

// Get the properties to extract from a resource.
// Uses mergedTransformConfig which contains default config merged with CollectorConfig CR customizations.
func getTransformConfig(group, kind string) (ResourceConfig, bool) {
	// Use mergedTransformConfig which is populated by LoadAndMergeConfigurableCollection
	// Falls back to defaultTransformConfig if configurable collection is disabled.
	// RLock scope is narrow: only the pointer copy needs protection. After this,
	// the local transformConfig points to an immutable map (never mutated after swap).
	configMu.RLock()
	transformConfig := mergedTransformConfig
	configMu.RUnlock()
	if transformConfig == nil {
		// Safety fallback if LoadAndMergeConfigurableCollection hasn't been called yet
		transformConfig = defaultTransformConfig
	}

	var val ResourceConfig
	var found bool

	if group == "" { // kubernetes core api resources
		val, found = transformConfig[kind]
	} else {
		val, found = transformConfig[kind+"."+group]
	}

	return val, found
}
