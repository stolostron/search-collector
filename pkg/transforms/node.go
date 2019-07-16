/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type NodeResource struct {
	*v1.Node
}

func (n NodeResource) BuildNode() Node {
	node := transformCommon(n) // Start off with the common properties

	var roles []string
	labels := n.ObjectMeta.Labels
	roleSet := map[string]struct{}{
		"node-role.kubernetes.io/proxy":      {},
		"node-role.kubernetes.io/management": {},
		"node-role.kubernetes.io/master":     {},
		"node-role.kubernetes.io/va":         {},
		"node-role.kubernetes.io/etcd":       {},
		"node-role.kubernetes.io/worker":     {},
	}

	for key, value := range labels {
		if _, found := roleSet[key]; found && value == "true" {
			roles = append(roles, strings.TrimPrefix(key, "node-role.kubernetes.io/"))
		}
	}

	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	// sort in alphabetical order to make the ui consistant
	sort.Strings(roles)

	// Extract the properties specific to this type
	node.Properties["kind"] = "Node"
	node.Properties["architecture"] = n.Status.NodeInfo.Architecture
	node.Properties["cpu"], _ = n.Status.Capacity.Cpu().AsInt64()
	node.Properties["osImage"] = n.Status.NodeInfo.OSImage
	node.Properties["role"] = roles

	return node
}

func (n NodeResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
