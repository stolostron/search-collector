package transforms

// Declares a property to extract from a resource using jsonpath.
type ExtractProperty struct {
	name     string // `json:"name,omitempty"`
	jsonpath string // `json:"jsonpath,omitempty"`
}

// Declares the properties to extract from a given resource.
type ResourceConfig struct {
	properties []ExtractProperty // `json:"properties,omitempty"`
}

// Declares properties to extract from the resource by default.
var defaultTransformConfig = map[string]ResourceConfig{
	"ClusterServiceVersion.operators.coreos.com": ResourceConfig{
		properties: []ExtractProperty{
			ExtractProperty{name: "version", jsonpath: "{.spec.version}"},
			ExtractProperty{name: "display", jsonpath: "{.spec.displayName}"},
			ExtractProperty{name: "phase", jsonpath: "{.status.phase}"},
		},
	},
	"Subscription.operators.coreos.com": ResourceConfig{
		properties: []ExtractProperty{
			ExtractProperty{name: "source", jsonpath: "{.spec.source}"},
			ExtractProperty{name: "package", jsonpath: "{.spec.name}"},
			ExtractProperty{name: "channel", jsonpath: "{.spec.channel}"},
			ExtractProperty{name: "installplan", jsonpath: "{.status.installedCSV}"},
			ExtractProperty{name: "phase", jsonpath: "{.status.state}"},
		},
	},
	"ClusterOperator.config.openshift.io": ResourceConfig{
		properties: []ExtractProperty{
			ExtractProperty{name: "version", jsonpath: `{.status.versions[?(@.name=="operator")].version}`},
			ExtractProperty{name: "available", jsonpath: `{.status.conditions[?(@.type=="Available")].status}`},
			ExtractProperty{name: "progressing", jsonpath: `{.status.conditions[?(@.type=="Progressing")].status}`},
			ExtractProperty{name: "degraded", jsonpath: `{.status.conditions[?(@.type=="Degraded")].status}`},
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
