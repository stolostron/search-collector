package transforms

import (
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
		if _, found := roleSet["key"]; found && value == "true" {
			roles = append(roles, strings.TrimPrefix(key, "node-role.kubernetes.io/"))
		}
	}

	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	// Extract the properties specific to this type
	node.Properties["kind"] = "Node"
	node.Properties["architecture"] = resource.Status.NodeInfo.Architecture
	node.Properties["cpu"] = resource.Status.Capacity.Cpu
	node.Properties["osImage"] = resource.Status.NodeInfo.OSImage
	node.Properties["role"] = roles

	// Form these properties into an rg.Node
	return node
}
