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

type DeployableResource struct {
	*mcm.Deployable
}

func (d DeployableResource) BuildNode() Node {
	node := transformCommon(d) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "Deployable"
	node.Properties["apigroup"] = "mcm.ibm.com"
	node.Properties["deployerKind"] = string(d.Spec.Deployer.DeployerKind)

	node.Properties["chartUrl"] = ""
	node.Properties["deployerNamespace"] = ""
	if d.Spec.Deployer.HelmDeployer != nil {
		node.Properties["chartUrl"] = d.Spec.Deployer.HelmDeployer.ChartURL
		node.Properties["deployerNamespace"] = d.Spec.Deployer.HelmDeployer.Namespace
	}

	return node
}

func (d DeployableResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
