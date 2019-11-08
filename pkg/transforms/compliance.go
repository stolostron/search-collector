/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	com "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/compliance/v1alpha1"
	policy "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"
)

type ComplianceResource struct {
	*com.Compliance
}

func (c ComplianceResource) BuildNode() Node {
	node := transformCommon(c)               // Start off with the common properties
	apiGroupVersion(c.TypeMeta, &node) // add kind, apigroup and version

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

	for _, comStatus := range c.Status.Status {
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

	node.Properties["policyCompliant"] = int64(policyCompliant)
	node.Properties["policyTotal"] = int64(policyTotal)
	node.Properties["clusterCompliant"] = int64(clusterCompliant)
	node.Properties["clusterTotal"] = int64(clusterTotal)

	if policyCompliant == policyTotal {
		node.Properties["status"] = "compliant"
	} else {
		node.Properties["status"] = "noncompliant"
	}

	return node
}

func (c ComplianceResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
