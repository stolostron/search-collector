/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Namespace and yields a Node
func transformNamespace(resource *v1.Namespace) Node {

	namespace := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	namespace.Properties["kind"] = "Namespace"
	namespace.Properties["status"] = string(resource.Status.Phase)

	return namespace
}
