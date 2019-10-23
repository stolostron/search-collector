/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strconv"
	"strings"

	"github.com/golang/glog"
	v1 "github.ibm.com/IBMPrivateCloud/ma-mcm-controller/pkg/apis/mcm/v1alpha1"
)

type MutationPolicyResource struct {
	*v1.MutationPolicy
}

func (m MutationPolicyResource) BuildNode() Node {
	node := transformCommon(m)
	apiGroupVersion(m.TypeMeta, &node) // add kind, apigroup and version

	// Extract the properties specific to this type
	node.Properties["compliant"] = string(m.Status.ComplianceState)
	var totalResources int
	for _, oval := range m.Status.CompliancyDetails {
		/*
			We are parsing a map[string]map[string][]string object m.Status.CompliancyDetails
			Here is an example
				"mutation-policy-example": {
			        "default": [
			                    "2 mutated pods detected in namespace `default`"
			                ],
			        "kube-public": [
			                    "3 mutated pods detected in namespace `kube-public`"
			                ]
			            }
		*/
		for _, ival := range oval {
			for _, str := range ival {
				substr := strings.Split(str, " ")
				podCount, err := strconv.Atoi(substr[0])
				if err != nil {
					glog.Warning("Parsing error in Compliance Details : Violated pod count may be wrong")
				} else {
					totalResources = totalResources + podCount
				}

			}
		}
	}
	node.Properties["mutatedResources"] = totalResources
	node.Properties["remediationAction"] = string(m.Spec.RemediationAction)
	node.Properties["severity"] = m.Spec.Severity
	return node
}

func (m MutationPolicyResource) BuildEdges(ns NodeStore) []Edge {
	return []Edge{}
}
