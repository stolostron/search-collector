package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
)

// Takes a *mcm.Deployable and yields a Node
func transformDeployable(resource *mcm.Deployable) Node {

	deployable := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	deployable.Properties["kind"] = "Deployable"
	deployable.Properties["deployerKind"] = string(resource.Spec.Deployer.DeployerKind)

	deployable.Properties["chartUrl"] = ""
	deployable.Properties["deployerNamespace"] = ""
	if resource.Spec.Deployer.HelmDeployer != nil {
		deployable.Properties["chartUrl"] = resource.Spec.Deployer.HelmDeployer.ChartURL
		deployable.Properties["deployerNamespace"] = resource.Spec.Deployer.HelmDeployer.Namespace
	}

	return deployable
}
