/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"strings"

	app "github.com/open-cluster-management/multicloud-operators-subscription-release/pkg/apis/apps/v1"
)

// AppHelmCRResource ...
type AppHelmCRResource struct {
	node Node
	Repo app.HelmReleaseRepo
}

// AppHelmCRResourceBuilder ...
func AppHelmCRResourceBuilder(a *app.HelmRelease) *AppHelmCRResource {
	node := transformCommon(a)         // Start off with the common properties
	apiGroupVersion(a.TypeMeta, &node) // add kind, apigroup and version

	// Add other properties
	if a.Repo.Source != nil && a.Repo.Source.SourceType != "" {
		node.Properties["sourceType"] = a.Repo.Source.SourceType
		sourceType := string(a.Repo.Source.SourceType)
		if strings.EqualFold(sourceType, "github") {
			node.Properties["url"] = a.Repo.Source.GitHub.Urls
			node.Properties["chartPath"] = a.Repo.Source.GitHub.ChartPath
			node.Properties["branch"] = a.Repo.Source.GitHub.Branch
		} else if strings.EqualFold(sourceType, "git") {
			node.Properties["url"] = a.Repo.Source.Git.Urls
			node.Properties["chartPath"] = a.Repo.Source.Git.ChartPath
			node.Properties["branch"] = a.Repo.Source.Git.Branch
		} else if strings.EqualFold(sourceType, "HelmRepo") {
			node.Properties["url"] = a.Repo.Source.HelmRepo.Urls
		}
	}

	// Need to pass repo so we can access it when building the edges.
	return &AppHelmCRResource{node: node, Repo: a.Repo}
}

// BuildNode construct the node for the AppHelmCRResource Resources
func (a AppHelmCRResource) BuildNode() Node {
	return a.node
}

// BuildEdges construct the edges for the AppHelmCRResource Resources
func (a AppHelmCRResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := a.node.UID

	nodeInfo := NodeInfo{
		NameSpace: a.node.Properties["namespace"].(string),
		UID:       UID,
		Kind:      a.node.Properties["kind"].(string),
		Name:      a.node.Properties["name"].(string),
		EdgeType:  "attachedTo"}

	// attachedTo edges
	releaseMap := make(map[string]struct{})

	if a.node.Properties["name"] != "" {
		releaseMap[a.node.Properties["name"].(string)] = struct{}{}
		ret = append(ret, edgesByDestinationName(releaseMap, "Release", nodeInfo, ns, []string{})...)
	}

	if a.Repo.SecretRef != nil {
		secretMap := make(map[string]struct{})
		if a.Repo.SecretRef.Name != "" {
			secretMap[a.Repo.SecretRef.Name] = struct{}{}
			ret = append(ret, edgesByDestinationName(secretMap, "Secret", nodeInfo, ns, []string{})...)
		}
	}

	if a.Repo.ConfigMapRef != nil {
		configmapMap := make(map[string]struct{})
		if a.Repo.ConfigMapRef.Name != "" {
			configmapMap[a.Repo.ConfigMapRef.Name] = struct{}{}
			ret = append(ret, edgesByDestinationName(configmapMap, "ConfigMap", nodeInfo, ns, []string{})...)
		}
	}
	return ret
}
