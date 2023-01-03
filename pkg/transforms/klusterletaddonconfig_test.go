/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
)

func TestTransformKlusterletAddonConfig(t *testing.T) {
	var p agentv1.KlusterletAddonConfig
	UnmarshalFile("klusterletaddonconfig.json", &p, t)
	node := KlusterletAddonConfigResourceBuilder(&p).BuildNode()

	enabledAddons := map[string]interface{}{
		"search-collector":       true,
		"policy-controller":      true,
		"cert-policy-controller": true,
		"application-manager":    true,
		"iam-policy-controller":  true,
	}

	addons := node.Properties["addon"].(map[string]interface{})
	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("name", node.Properties["name"], "sample-kac", t)
	AssertEqual("namespace", node.Properties["namespace"], "sample-kac", t)
	AssertEqual("addon", len(addons), len(enabledAddons), t)
	AssertEqual("searchAddon", addons["search-collector"], false, t)
	AssertEqual("policyControllerAddon", addons["policy-controller"], true, t)
}
