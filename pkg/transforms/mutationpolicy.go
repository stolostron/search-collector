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
	ret := []Edge{}
	UID := prefixedUID(m.UID)
	currentMANode := ns.ByUID[UID]
	if currentMANode.Properties["compliant"] != "NonCompliant" {
		return ret //We need to build edges only if the MAPolicy is not compliant
	}
	for _, event := range ns.K8sEventNodes {

		policyUID, iObjectPresent := event.Node.Properties["InvolvedObject.uid"].(string)
		resourceUID, muidPresent := event.Node.Properties["message.uid"].(string)
		if !iObjectPresent || !muidPresent {
			continue // if there is no useful values in Event skip it
		}
		policyInEvent := prefixedUIDStr(policyUID)
		vulnerableResourceInEvent := prefixedUIDStr(resourceUID)
		if policyInEvent != UID { // If the event does not speak about current Vulnerability skip it
			continue
		} //Else
		//Check if the resources are present in our system before creating Edges
		_, goodResource := ns.ByUID[vulnerableResourceInEvent]
		if !(goodResource) {
			glog.V(2).Infof("Resource %s not found - No Edge Created", vulnerableResourceInEvent)
			continue // Resource should be in our Node list to make a edge
		}
		edgeVal := Edge{
			EdgeType:  "violates",
			SourceUID: vulnerableResourceInEvent,
			DestUID:   UID,
		}
		ret = append(ret, edgeVal)
		remoteSubscription := getSubscriptionByUID(vulnerableResourceInEvent, ns)
		glog.V(4).Infof("Found subscription %s attached to resource in violation ", remoteSubscription)
		if len(remoteSubscription) > 0 {
			subEdge := Edge{
				EdgeType:  "violates",
				SourceUID: UID,
				DestUID:   remoteSubscription,
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
