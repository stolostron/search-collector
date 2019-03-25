package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

// Takes a *mcm.ApplicationRelationship and yields a Node
func transformApplicationRelationship(resource *mcm.ApplicationRelationship) Node {
	aR := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	aR.Properties["kind"] = "ApplicationRelationship"
	aR.Properties["destination"] = resource.Spec.Destination.Name
	aR.Properties["source"] = resource.Spec.Source.Name
	aR.Properties["type"] = string(resource.Spec.RelType)

	return aR
}
