package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Service and yields a Node
func transformService(resource *v1.Service) Node {

	service := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	service.Properties["kind"] = "Service"

	return service
}
