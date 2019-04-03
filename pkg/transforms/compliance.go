package transforms

import (
	com "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/compliance/v1alpha1"
	policy "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"
)

// Takes a *mcm.Compliance and yields a Node
func transformCompliance(resource *com.Compliance) Node {

	compliance := transformCommon(resource) // Start off with the common properties
	compliance.Properties["kind"] = "Compliance"

	// On a Compliance object, status.status holds an object with a property for each cluster.
	// First loop through and check cluster object statuses
	// Then each of those cluster objects hold property objects, which say whether the cluster is compliant to that policy.
	// A policy is said to be itself "NonCompliant" if and only if one or more of the clusters the policy applies to is noncompliant.
	// So, we keep the policies in a map (an object) and change their overall value based on what we find in the cluster specific entries.
	// After that, we loop back through this map of policies and count, because what we want to return is 2 pairs of numbers -
	// These pairs are (A,B) and (C,D) where A out of B clusters are compliant to all of the policies the compliance places on them,
	// and C out of D policies are complied to by all clusters to which they apply.
	policyCompliant := 0
	policyTotal := 0
	clusterCompliant := 0
	clusterTotal := 0

	for _, comStatus := range resource.Status.Status {
		clusterTotal++
		if comStatus.ComplianceState == policy.Compliant {
			clusterCompliant++
		}

		for _, aggStatus := range comStatus.AggregatePolicyStatus {
			policyTotal++
			if aggStatus.ComplianceState == policy.Compliant {
				policyCompliant++
			}
		}
	}

	compliance.Properties["policyCompliant"] = int64(policyCompliant)
	compliance.Properties["policyTotal"] = int64(policyTotal)
	compliance.Properties["clusterCompliant"] = int64(clusterCompliant)
	compliance.Properties["clusterTotal"] = int64(clusterTotal)

	if policyCompliant == policyTotal {
		compliance.Properties["status"] = "compliant"
	} else {
		compliance.Properties["status"] = "noncompliant"
	}

	return compliance
}
