package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Secret and yields a Node
func TransformSecret(resource *v1.Secret) Node {

	secret := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	secret.Properties["kind"] = "Secret"

	return secret
}
