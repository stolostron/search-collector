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
func transformDeployable(resource *mcm.Deployable) Node {

	deployable := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	deployable.Properties["kind"] = "Deployable"
	deployable.Properties["apigroup"] = "mcm.ibm.com"
	deployable.Properties["deployerKind"] = string(resource.Spec.Deployer.DeployerKind)

	deployable.Properties["chartUrl"] = ""
	deployable.Properties["deployerNamespace"] = ""
	if resource.Spec.Deployer.HelmDeployer != nil {
		deployable.Properties["chartUrl"] = resource.Spec.Deployer.HelmDeployer.ChartURL
		deployable.Properties["deployerNamespace"] = resource.Spec.Deployer.HelmDeployer.Namespace
	}

	return deployable
}
