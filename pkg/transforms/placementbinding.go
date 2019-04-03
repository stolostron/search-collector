/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"fmt"

	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

// Takes a *mcm.PlacementBinding and yields a Node
func transformPlacementBinding(resource *mcm.PlacementBinding) Node {

	placementBinding := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	placementBinding.Properties["kind"] = "PlacementBinding"
	placementBinding.Properties["apigroup"] = "mcm.ibm.com"

	name := resource.PlacementPolicyRef.Name
	kind := resource.PlacementPolicyRef.Kind
	placementBinding.Properties["placementpolicy"] = fmt.Sprintf("%s (%s)", name, kind)

	l := len(resource.Subjects)
	subjects := make([]string, l)
	for i := 0; i < l; i++ {
		name := resource.Subjects[i].Name
		kind := resource.Subjects[i].Kind
		subjects[i] = fmt.Sprintf("%s (%s)", name, kind)
	}
	placementBinding.Properties["subject"] = subjects

	return placementBinding
}
