/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
*/

package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func TestTransformHelmRelease(t *testing.T) {
	var c v1.ConfigMap
	var r release.Release

	UnmarshalFile("../../test-data/helmrelease-configmap.json", &c, t)
	UnmarshalFile("../../test-data/helmrelease-release.json", &r, t)

	node := HelmReleaseResource{&c, &r}.BuildNode()

	// Test only the fields that exist in HelmRelease - the common test will test the other bits
	AssertEqual("name", node.Properties["name"], "helmrelease-ex", t)
	AssertEqual("kind", node.Properties["kind"], "Release", t)
	AssertEqual("status", node.Properties["status"], "DEPLOYED", t)
	AssertEqual("revision", node.Properties["revision"], int64(1), t)
	AssertEqual("chartName", node.Properties["chartName"], "ibm-nodejs-sample", t)
	AssertEqual("chartVersion", node.Properties["chartVersion"], "2.0.0", t)
	AssertEqual("namespace", node.Properties["namespace"], "default", t)
	AssertEqual("updated", node.Properties["updated"], "2019-07-18T14:58:37Z", t)
	AssertEqual("_hubClusterResource", node.Properties["_hubClusterResource"], true, t)
	AssertEqual("_clusterNamespace", node.Properties["_clusterNamespace"], nil, t)

}
