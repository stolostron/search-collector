package transforms

// Declares properties to extract from the resource by default.
var defaultTransformConfig = map[string]ResourceConfig{
	"clusterserviceversion.operators.coreos.com": ResourceConfig{
		// apigroup: "operators.coreos.com",
		// kind:     "ClusterServiceVersion",
		properties: []ExtractProperty{
			ExtractProperty{propType: "string", name: "version", path: []string{"spec", "version"}},
			ExtractProperty{propType: "string", name: "display", path: []string{"spec", "displayName"}},
			ExtractProperty{propType: "string", name: "phase", path: []string{"status", "phase"}},
		},
	},
	"subscriptions.operators.coreos.com": ResourceConfig{
		// apigroup: "operators.coreos.com",
		// kind:     "Subscription",
		properties: []ExtractProperty{
			ExtractProperty{propType: "string", name: "source", path: []string{"spec", "source"}},
			ExtractProperty{propType: "string", name: "package", path: []string{"spec", "name"}},
			ExtractProperty{propType: "string", name: "channel", path: []string{"spec", "channel"}},
			ExtractProperty{propType: "string", name: "installplan", path: []string{"status", "installedCSV"}},
			ExtractProperty{propType: "string", name: "phase", path: []string{"status", "state"}},
		},
	},
}

// Get which properties to extract from a resource.
func getTransformConfig(group, kind string) ResourceConfig {
	transformConfig := defaultTransformConfig

	// FUTURE: We want to create this dynamically by reading from:
	// 	  1. ConfigMap where it can be customized by the users.
	// 	  2. CRD "additionalPrinterColumns" field.

	if val, ok := transformConfig[kind+"."+group]; ok {
		return val
	}
	return ResourceConfig{}
}

// "clusteroperator.config.openshift.io": ResourceConfig{
// 	// apigroup: "config.openshift.io",
// 	// kind:     "ClusterOperator",
// 	properties: []ExtractProperty{
// 		ExtractProperty{propType: "string", name: "version", path: []string{"status", "versions", "name"}},

// 		// - additionalPrinterColumns:
// 		// - description: The version the operator is at.
// 		//   jsonPath: .status.versions[?(@.name=="operator")].version
// 		//   name: Version
// 		//   type: string
// 		// - description: Whether the operator is running and stable.
// 		//   jsonPath: .status.conditions[?(@.type=="Available")].status
// 		//   name: Available
// 		//   type: string
// 		// - description: Whether the operator is processing changes.
// 		//   jsonPath: .status.conditions[?(@.type=="Progressing")].status
// 		//   name: Progressing
// 		//   type: string
// 		// - description: Whether the operator is degraded.
// 		//   jsonPath: .status.conditions[?(@.type=="Degraded")].status
// 		//   name: Degraded
// 		//   type: string
// 		// - description: The time the operator's Available status last changed.
// 		//   jsonPath: .status.conditions[?(@.type=="Available")].lastTransitionTime
// 		//   name: Since
// 		//   type: date
// 	},