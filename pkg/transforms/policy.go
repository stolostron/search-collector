/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	mcm "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"
)

// Takes a *mcm.Policy and yields a Node
func transformPolicy(resource *mcm.Policy) Node {

	policy := transformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	policy.Properties["kind"] = "Policy"
	policy.Properties["apigroup"] = "policy.mcm.ibm.com"
	policy.Properties["remediationAction"] = string(resource.Spec.RemediationAction)
	policy.Properties["compliant"] = string(resource.Status.ComplianceState)
	policy.Properties["valid"] = resource.Status.Valid

	rules := int64(0)
	if resource.Spec.RoleTemplates != nil {
		for _, role := range resource.Spec.RoleTemplates {
			if role != nil {
				rules += int64(len(role.Rules))
			}
		}
	}
	policy.Properties["numRules"] = rules

	return policy
}
