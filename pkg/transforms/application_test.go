package transforms

import (
	"testing"

	v1 "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
)

func TestTransformApplication(t *testing.T) {
	var a v1.Application
	UnmarshalFile("../../test-data/application.json", &a, t)
	node := transformApplication(&a)

	// Test only the fields that exist in application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
	AssertEqual("dashboard", node.Properties["dashboard"], "https://0.0.0.0:8443/grafana/dashboard/test", t)
}
