package transforms

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	Name     string // `json:"name,omitempty"`
	JSONPath string // `json:"jsonpath,omitempty"`
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
	"Subscription.operators.coreos.com": {
		properties: []ExtractProperty{
			{Name: "source", JSONPath: "{.spec.source}"},
			{Name: "package", JSONPath: "{.spec.name}"},
			{Name: "channel", JSONPath: "{.spec.channel}"},
			{Name: "installplan", JSONPath: "{.status.installedCSV}"},
			{Name: "phase", JSONPath: "{.status.state}"},
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
	"VirtualMachine.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "cpu", JSONPath: `{.spec.template.spec.domain.cpu.cores}`},
			{Name: "memory", JSONPath: `{.spec.template.spec.domain.memory.guest}`},
			{Name: "status", JSONPath: `{.status.printableStatus}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
		},
	},
	"VirtualMachineInstance.kubevirt.io": {
		properties: []ExtractProperty{
			{Name: "node", JSONPath: `{.status.nodeName}`},
			{Name: "ipaddress", JSONPath: `{.status.interfaces[0].ipAddress}`},
			{Name: "phase", JSONPath: `{.status.phase}`},
			{Name: "ready", JSONPath: `{.status.conditions[?(@.type=='Ready')].status}`},
			{Name: "liveMigratable", JSONPath: `{.status.conditions[?(@.type=='LiveMigratable')].status}`},
		},
	},
}

// Get the properties to extract from a resource.
func getTransformConfig(group, kind string) (ResourceConfig, bool) {
	transformConfig := defaultTransformConfig

	// FUTURE: We want to create this dynamically by reading from:
	// 	  1. ConfigMap where it can be customized by the users.
	// 	  2. CRD "additionalPrinterColumns" field.

	val, found := transformConfig[kind+"."+group]
	return val, found
}
