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
