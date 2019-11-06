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
	var podUIDTexts []string
	var allPodUids string
	for _, oval := range m.Status.CompliancyDetails {
		/*
						We are parsing a map[string]map[string][]string object m.Status.CompliancyDetails
						Here is an example
							"mutation-policy-example": {
						        "default": [
			                    "2 mutated pods detected in namespace `default`:[98b3c272-fbff-11e9-aa82-00163e01bcd9,3f4f13f2-f900-11e9-aa82-00163e01bcd9]"
			                ],
			                "kube-public": [
			                    "0 mutated pods detected in namespace `kube-public`:[]"
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
				cutLeft := strings.SplitAfter(str, "[")        // TrimLeft in Golang has a issue with char [ so using SplitAfter
				cutRight := strings.TrimRight(cutLeft[1], "]") // Get whats inbetween []
				if len(cutRight) > 0 {
					//If there is text
					podUIDTexts = append(podUIDTexts, cutRight)
				}
			}
		}
	}
	allPodUids = strings.Join(podUIDTexts, ",")
	node.Properties["mutatedResources"] = totalResources
	node.Properties["remediationAction"] = string(m.Spec.RemediationAction)
	node.Properties["severity"] = m.Spec.Severity
	node.Metadata["_mutatedUIDs"] = allPodUids
	return node
}

func (m MutationPolicyResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(m.UID)
	currentMANode := ns.ByUID[UID]
	podUIDs := strings.Split(currentMANode.GetMetadata("_mutatedUIDs"), ",")
	if currentMANode.Properties["compliant"] != "NonCompliant" || len(podUIDs) == 0 {
		return ret //We need to build edges only if the MAPolicy is not compliant , Or there is no UIDs in status to connect
	}
	for _, resourceUID := range podUIDs {
		vulnerableResource := prefixedUIDStr(resourceUID)
		//Check if the resources are present in our system before creating Edges
		_, goodResource := ns.ByUID[vulnerableResource]
		if !(goodResource) {
			glog.V(2).Infof("Resource %s not found - No Edge Created", vulnerableResource)
			continue // Resource should be in our Node list to make a edge
		}

		edgeVal := Edge{
			EdgeType:  "violates",
			SourceUID: vulnerableResource,
			DestUID:   UID,
		}
		ret = append(ret, edgeVal)
		remoteSubscription := getSubscriptionByUID(vulnerableResource, ns)
		glog.V(4).Infof("Found subscription %s attached to resource in violation ", remoteSubscription)
		if len(remoteSubscription) > 0 {
			subEdge := Edge{
				EdgeType:  "violates",
				SourceUID: remoteSubscription,
				DestUID:   UID,
			}
			ret = append(ret, subEdge)
		}
	}
	nodeInfo := NodeInfo{Name: m.Name, NameSpace: m.Namespace, UID: UID, EdgeType: "ownedBy", Kind: m.Kind}
	//ownedBy edges
	if currentMANode.GetMetadata("OwnerUID") != "" {
		ret = append(ret, edgesByOwner(currentMANode.GetMetadata("OwnerUID"), ns, nodeInfo)...)
	}
	return ret
}
