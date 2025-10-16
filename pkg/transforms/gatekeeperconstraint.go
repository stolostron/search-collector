// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type GkConstraintResource struct {
	node Node
}

func GkConstraintResourceBuilder(
	c *unstructured.Unstructured, additionalColumns ...ExtractProperty,
) *GkConstraintResource {
	n := transformCommon(c)
	n = applyDefaultTransformConfig(n, c, additionalColumns...)

	typeMeta := metav1.TypeMeta{
		Kind:       c.GetKind(),
		APIVersion: c.GetAPIVersion(),
	}

	apiGroupVersion(typeMeta, &n) // add kind, apigroup and version

	n.Properties["_isExternal"] = getIsPolicyExternal(c)

	totalViolations, found, err := unstructured.NestedFieldNoCopy(c.Object, "status", "totalViolations")
	if found && err == nil {
		if fmt.Sprint(totalViolations) == "0" { // numbers in JSON/YAML can be strange
			n.Properties["compliant"] = "Compliant"
		} else {
			n.Properties["compliant"] = "NonCompliant"
		}
	}

	n = recordConstraintViolations(c, n)

	return &GkConstraintResource{node: n}
}

func (r GkConstraintResource) BuildNode() Node {
	return r.node
}

func (r GkConstraintResource) BuildEdges(ns NodeStore) []Edge {
	constraintKind, ok := r.node.Properties["kind"].(string)
	if !ok {
		return []Edge{}
	}

	relObjs, ok := r.node.Metadata["relObjs"].([]relatedObject)
	if !ok {
		return []Edge{}
	}

	edges := make([]Edge, 0, len(relObjs))

	for _, obj := range relObjs {
		namespace := obj.Namespace
		if namespace == "" {
			namespace = "_NONE"
		}

		// ignore objects if they aren't in the NodeStore
		if res, ok := ns.ByKindNamespaceName[obj.Kind][namespace][obj.Name]; ok {
			edges = append(edges, Edge{
				EdgeType:   obj.EdgeType,
				SourceKind: constraintKind,
				SourceUID:  r.node.UID,
				DestKind:   obj.Kind,
				DestUID:    res.UID,
			})
		}
	}

	return edges
}

func recordConstraintViolations(c *unstructured.Unstructured, node Node) Node {
	violations, found, err := unstructured.NestedSlice(c.Object, "status", "violations")
	if !found || err != nil {
		return node
	}

	// Use a set to remove possible duplicate resources (resources can have multiple violations)
	objSet := make(map[relatedObject]bool)

	for _, item := range violations {
		relObj := parseGkConstraintViolation(item)
		if relObj != nil {
			objSet[*relObj] = true
		}
	}

	objList := make([]relatedObject, 0, len(objSet))

	for obj := range objSet {
		objList = append(objList, obj)
	}

	slices.SortFunc(objList, func(a, b relatedObject) int {
		return strings.Compare(a.String(), b.String())
	})

	node.Metadata["relObjs"] = objList

	return node
}

func parseGkConstraintViolation(item any) *relatedObject {
	violation, ok := item.(map[string]any)
	if !ok {
		return nil
	}

	group, found, err := unstructured.NestedString(violation, "group")
	if !found || err != nil {
		return nil
	}

	kind, found, err := unstructured.NestedString(violation, "kind")
	if !found || err != nil {
		return nil
	}

	name, found, err := unstructured.NestedString(violation, "name")
	if !found || err != nil {
		return nil
	}

	// cluster-scoped objects will have no namespace, and it will be an empty string here
	namespace, _, err := unstructured.NestedString(violation, "namespace")
	if err != nil {
		return nil
	}

	version, found, err := unstructured.NestedString(violation, "version")
	if !found || err != nil {
		return nil
	}

	return &relatedObject{
		Group:     group,
		Version:   version,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		EdgeType:  noncompliantEdge, // GK only reports violations, never compliant objects.
	}
}
