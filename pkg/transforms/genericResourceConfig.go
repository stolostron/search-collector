package transforms

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	Name     string // `json:"name,omitempty"`
	JSONPath string // `json:"jsonpath,omitempty"`
	// Denotes this property represents a memory field of a resource and must be converted to bytes for standardization
	isMemory bool
	// An internal property to denote this property should be set on the node's metadata instead.
	metadataOnly bool
}

// Declares the properties to extract from a given resource.
type ResourceConfig struct {
	properties []ExtractProperty // `json:"properties,omitempty"`
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
			{Name: "memoryAllocatable", JSONPath: `{.status.allocatable.memory}`, isMemory: true},
			{Name: "memoryCapacity", JSONPath: `{.status.capacity.memory}`, isMemory: true},
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
			{Name: "agentConnected", JSONPath: `{.status.conditions[?(@.type=="AgentConnected")].status}`},
			{Name: "cpu", JSONPath: `{.spec.template.spec.domain.cpu.cores}`},
			{Name: "flavor", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/flavor}`},
			{Name: "memory", JSONPath: `{.spec.template.spec.domain.memory.guest}`, isMemory: true},
			{Name: "osName", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/os}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "status", JSONPath: `{.status.printableStatus}`},
			{Name: "workload", JSONPath: `{.spec.template.metadata.annotations.\vm\.kubevirt\.io/workload}`},
			{Name: "_specRunning", JSONPath: `{.spec.running}`},
			{Name: "_specRunStrategy", JSONPath: `{.spec.runStrategy}`},
		},
	},
	"VirtualMachineInstance.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ipaddress", JSONPath: `{.status.interfaces[0].ipAddress}`},
			{Name: "liveMigratable", JSONPath: `{.status.conditions[?(@.type=='LiveMigratable')].status}`},
			{Name: "node", JSONPath: `{.status.nodeName}`},
			{Name: "osVersion", JSONPath: `{.status.guestOSInfo.version}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "vmSize", JSONPath: `{.metadata.labels.\kubevirt\.io/size}`},
		},
	},
	"VirtualMachineSnapshot.snapshot.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "status", JSONPath: `{.status.conditions[?(@.type=='Ready')].reason}`},
			{Name: "indications", JSONPath: `{.status.indications}`}, // this is an array of strings - will collect array items separated by ;
			{Name: "sourceVM", JSONPath: `{.spec.source.name}`},
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
