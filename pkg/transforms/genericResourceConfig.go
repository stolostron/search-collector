package transforms

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	Name     string   // `json:"name,omitempty"`
	JSONPath string   // `json:"jsonpath,omitempty"`
	DataType DataType // `json:"dataType,omitempty"`
	// An internal property to denote this property should be set on the node's metadata instead.
	metadataOnly bool
}

type ExtractEdge struct {
	Name   string   // `json:"name,omitempty"`
	ToKind string   // `json:"toKind,omitempty"`
	Type   EdgeType // `json:"type,omitempty"`
}

type DataType string

const (
	DataTypeBytes  DataType = "bytes"
	DataTypeSlice  DataType = "slice"
	DataTypeString DataType = "string"
	DataTypeNumber DataType = "number"
)

// Declares the properties to extract from a given resource.
type ResourceConfig struct {
	properties []ExtractProperty // `json:"properties,omitempty"`
	edges      []ExtractEdge     // `json:"edges,omitempty"`
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

// Declares properties to extract from the resource by default.
var defaultTransformConfig = map[string]ResourceConfig{
	"ClusterServiceVersion.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: "{.spec.version}"},
			{Name: "display", JSONPath: "{.spec.displayName}"},
			{Name: "phase", JSONPath: "{.status.phase}"},
		},
	},
	"ClusterOperator.config.openshift.io": {
		properties: []ExtractProperty{
			{Name: "version", JSONPath: `{.status.versions[?(@.name=="operator")].version}`},
			{Name: "available", JSONPath: `{.status.conditions[?(@.type=="Available")].status}`},
			{Name: "progressing", JSONPath: `{.status.conditions[?(@.type=="Progressing")].status}`},
			{Name: "degraded", JSONPath: `{.status.conditions[?(@.type=="Degraded")].status}`},
		},
	},
	"DataVolume.cdi.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "size", JSONPath: `{.spec.storage.resources.requests.storage}`},
			{Name: "storageClassName", JSONPath: `{.spec.storage.storageClassName}`},
		},
	},
	"Namespace": {
		properties: []ExtractProperty{
			{Name: "status", JSONPath: `{.status.phase}`},
		},
	},
	"Node": {
		properties: []ExtractProperty{
			{Name: "ipAddress", JSONPath: `{.status.addresses[?(@.type=="InternalIP")].address}`},
			{Name: "memoryAllocatable", JSONPath: `{.status.allocatable.memory}`, DataType: DataTypeBytes},
			{Name: "memoryCapacity", JSONPath: `{.status.capacity.memory}`, DataType: DataTypeBytes},
		},
	},
	"PersistentVolumeClaim": {
		properties: []ExtractProperty{
			{Name: "requestedStorage", JSONPath: `{.spec.resources.requests.storage}`, DataType: DataTypeBytes},
			{Name: "volumeMode", JSONPath: `{.spec.volumeMode}`},
		},
	},
	"StorageClass.storage.k8s.io": {
		properties: []ExtractProperty{
			{Name: "allowVolumeExpansion", JSONPath: `{.allowVolumeExpansion}`},
			{Name: "provisioner", JSONPath: `{.provisioner}`},
			{Name: "reclaimPolicy", JSONPath: `{.reclaimPolicy}`},
			{Name: "volumeBindingMode", JSONPath: `{.volumeBindingMode}`},
		},
	},
	"Subscription.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "source", JSONPath: "{.spec.source}"},
			{Name: "package", JSONPath: "{.spec.name}"},
			{Name: "channel", JSONPath: "{.spec.channel}"},
			{Name: "installplan", JSONPath: "{.status.installedCSV}"},
			{Name: "phase", JSONPath: "{.status.state}"},
		},
	},
	"ValidatingAdmissionPolicy.admissionregistration.k8s.io": {
		properties: []ExtractProperty{
			{Name: "paramKind_kind", JSONPath: `{.spec.paramKind.kind}`, metadataOnly: true},
			{Name: "paramKind_apiVersion", JSONPath: `{.spec.paramKind.apiVersion}`, metadataOnly: true},
		},
	},
	"VirtualMachine.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "architecture", JSONPath: `{.spec.template.spec.architecture}`},
			{Name: "agentConnected", JSONPath: `{.status.conditions[?(@.type=="AgentConnected")].status}`},
			{Name: "cpu", JSONPath: `{.spec.template.spec.domain.cpu.cores}`},
			{Name: "cpuSockets", JSONPath: `{.spec.template.spec.domain.cpu.sockets}`},
			{Name: "cpuThreads", JSONPath: `{.spec.template.spec.domain.cpu.threads}`},
			{Name: "dataVolumeNames", JSONPath: `{.spec.template.spec.volumes[*].dataVolume.name}`, DataType: DataTypeSlice},
			{Name: "_description", JSONPath: `{.metadata.annotations.description}`},
			{Name: "flavor", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/flavor}`},
			{Name: "memory", JSONPath: `{.spec.template.spec.domain.memory.guest}`, DataType: DataTypeBytes},
			{Name: "osName", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/os}`},
			{Name: "pvcClaimNames", JSONPath: `{.spec.template.spec.volumes[*].persistentVolumeClaim.claimName}`, DataType: DataTypeSlice},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "runStrategy", JSONPath: `{.spec.runStrategy}`},
			{Name: "status", JSONPath: `{.status.printableStatus}`},
			{Name: "_specRunning", JSONPath: `{.spec.running}`},
			{Name: "_specRunStrategy", JSONPath: `{.spec.runStrategy}`},
			{Name: "workload", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/workload}`},
		},
		edges: []ExtractEdge{
			{Name: "dataVolumeNames", ToKind: "DataVolume", Type: attachedTo},
			{Name: "pvcClaimNames", ToKind: "PersistentVolumeClaim", Type: attachedTo},
		},
	},
	"VirtualMachineInstance.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "architecture", JSONPath: `{.spec.architecture}`},
			{Name: "cpu", JSONPath: `{.spec.domain.cpu.cores}`},
			{Name: "cpuSockets", JSONPath: `{.spec.domain.cpu.sockets}`},
			{Name: "cpuThreads", JSONPath: `{.spec.domain.cpu.threads}`},
			{Name: "ipaddress", JSONPath: `{.status.interfaces[0].ipAddress}`},
			{Name: "liveMigratable", JSONPath: `{.status.conditions[?(@.type=='LiveMigratable')].status}`},
			{Name: "memory", JSONPath: `{.spec.domain.memory.guest}`, DataType: DataTypeBytes},
			{Name: "node", JSONPath: `{.status.nodeName}`},
			{Name: "osVersion", JSONPath: `{.status.guestOSInfo.version}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "vmSize", JSONPath: `{.metadata.labels.\kubevirt\.io/size}`},
		},
	},
	"VirtualMachineInstanceMigration.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "endTime", JSONPath: `{.status.migrationState.endTimestamp}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "vmiName", JSONPath: `{.spec.vmiName}`},
			{Name: "vmiName", JSONPath: `{.spec.vmiName}`, metadataOnly: true}, // Used to build the edge
		},
		edges: []ExtractEdge{
			{Name: "vmiName", ToKind: "VirtualMachineInstance", Type: migrationOf},
		},
	},
	"VirtualMachineSnapshot.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "_conditionReadyReason", JSONPath: `{.status.conditions[?(@.type=='Ready')].reason}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "indications", JSONPath: `{.status.indications}`}, // this is an array of strings - will collect array items separated by ;
			{Name: "sourceKind", JSONPath: `{.spec.source.kind}`},
			{Name: "sourceName", JSONPath: `{.spec.source.name}`},
			{Name: "readyToUse", JSONPath: `{.status.readyToUse}`},
		},
	},
	"VirtualMachineRestore.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "_conditionReadyReason", JSONPath: `{.status.conditions[?(@.type=='Ready')].reason}`},
			{Name: "restoreTime", JSONPath: `{.status.restoreTime}`},
			{Name: "complete", JSONPath: `{.status.complete}`},
			{Name: "targetKind", JSONPath: `{.spec.target.kind}`},
			{Name: "targetName", JSONPath: `{.spec.target.name}`},
		},
	},
}

// Get the properties to extract from a resource.
func getTransformConfig(group, kind string) (ResourceConfig, bool) {
	transformConfig := defaultTransformConfig

	// FUTURE: We want to create this dynamically by reading from:
	// 	  1. ConfigMap where it can be customized by the users.
	// 	  2. CRD "additionalPrinterColumns" field.

	var val ResourceConfig
	var found bool

	if group == "" { // kubernetes core api resources
		val, found = transformConfig[kind]
	} else {
		val, found = transformConfig[kind+"."+group]
	}

	return val, found
}
