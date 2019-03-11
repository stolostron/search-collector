package transforms

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.Node and yields a Node
func TransformNode(resource *v1.Node) Node {

	node := TransformCommon(resource) // Start off with the common properties

	var roles []string
	labels := resource.ObjectMeta.Labels
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
	node.Properties["architecture"] = resource.Status.NodeInfo.Architecture
	node.Properties["cpu"], _ = resource.Status.Capacity.Cpu().AsInt64()
	node.Properties["osImage"] = resource.Status.NodeInfo.OSImage
	node.Properties["role"] = strings.Join(roles, ", ")

	return node
}
