// Copyright Contributors to the Open Cluster Management project

package transforms

// ArgoApplicationResource ...
type ArgoApplicationResource struct {
	node Node
}

// ArgoApplicationResourceBuilder ...
func ArgoApplicationResourceBuilder(node Node, u map[string]interface{}) *ArgoApplicationResource {
	// Extract the properties specific to this type

	spec := u["spec"].(map[string]interface{})
	// Destination properties
	destination := spec["destination"].(map[string]interface{})
	if destination["name"] != "" {
		node.Properties["destinationName"] = destination["name"]
	}
	if destination["namespace"] != "" {
		node.Properties["destinationNamespace"] = destination["namespace"]
	}
	if destination["server"] != "" {
		node.Properties["destinationServer"] = destination["server"]
	}
	// Source properties
	source := spec["source"].(map[string]interface{})
	if source["path"] != "" {
		node.Properties["path"] = source["path"]
	}
	if source["chart"] != "" {
		node.Properties["chart"] = source["chart"]
	}
	if source["repoURL"] != "" {
		node.Properties["repoURL"] = source["repoURL"]
	}
	if source["targetRevision"] != "" {
		node.Properties["targetRevision"] = source["targetRevision"]
	}

	return &ArgoApplicationResource{node: node}
}

// BuildNode construct the node for the Application Resources
func (a ArgoApplicationResource) BuildNode() Node {
	return a.node
}

// BuildEdges construct the edges for the Application Resources
// See documentation at pkg/transforms/README.md
func (a ArgoApplicationResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	return ret
}
