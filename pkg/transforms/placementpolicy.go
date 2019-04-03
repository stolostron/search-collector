package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

// Takes a *mcm.PlacementPolicy and yields a Node
func transformPlacementPolicy(resource *mcm.PlacementPolicy) Node {

	placementPolicy := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	placementPolicy.Properties["kind"] = "PlacementPolicy"
	placementPolicy.Properties["apigroup"] = "mcm.ibm.com"

	placementPolicy.Properties["replicas"] = int64(0)
	if resource.Spec.ClusterReplicas != nil {
		placementPolicy.Properties["replicas"] = int64(*resource.Spec.ClusterReplicas)
	}

	l := len(resource.Status.Decisions)
	decisions := make([]string, l)
	for i := 0; i < l; i++ {
		decisions[i] = resource.Status.Decisions[i].ClusterName
	}
	placementPolicy.Properties["decisions"] = decisions

	return placementPolicy
}
