package transforms

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTransformNode(t *testing.T) {
	var n v1.Node
	UnmarshalFile("../../test-data/node.json", &n, t)
	node := transformNode(&n)

	// Test only the fields that exist in node - the common test will test the other bits
	AssertEqual("architecture", node.Properties["architecture"], "amd64", t)
	AssertEqual("cpu", node.Properties["cpu"], int64(8), t)
	AssertEqual("osImage", node.Properties["osImage"], "Ubuntu 16.04.5 LTS", t)
	AssertEqual("role", node.Properties["role"], "etcd, management, master, proxy, va", t)
}
