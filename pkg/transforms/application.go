/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

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
