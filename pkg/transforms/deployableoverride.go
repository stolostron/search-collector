/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

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
