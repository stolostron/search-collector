package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

// Takes a *mcm.Deployable and yields a Node
func transformDeployableOverride(resource *mcm.DeployableOverride) Node {

	dO := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	dO.Properties["kind"] = "DeployableOverride"
	dO.Properties["apigroup"] = "mcm.ibm.com"

	return dO
}
