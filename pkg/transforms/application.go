package transforms

import (
	v1 "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
)

// Takes a *mcm.Application and yields a Node
func transformApplication(resource *v1.Application) Node {

	application := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	application.Properties["kind"] = "Application"
	application.Properties["apigroup"] = "app.k8s.io"
	application.Properties["dashboard"] = resource.GetAnnotations()["apps.ibm.com/dashboard"]

	return application
}
