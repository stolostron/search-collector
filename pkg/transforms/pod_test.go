package transforms

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
)

// Helper to create a resource for use in testing.
func CreatePod(t *testing.T) *v1.Pod {
	var p v1.Pod

	rawPodBytes, err := ioutil.ReadFile("../../test-data/pod.json")
	if err != nil {
		t.Fatal("Unable to read test data", err)
	}
	err = json.Unmarshal(rawPodBytes, &p) // Stuff the unmarshalled data into p
	if err != nil {
		t.Fatal("Unable to unmarshal json to Pod", err)
	}

	return &p
}

func TestTransformPod(t *testing.T) {

	res := CreatePod(t)

	tp := TransformPod(res)

	// Build time struct matching time in test data
	date := time.Date(2019, 02, 21, 21, 30, 33, 0, time.UTC)

	// Test only the fields that exist in pods - the common test will test the other bits
	AssertEqual("kind", tp.Properties["kind"], "Pod", t)
	AssertEqual("hostIP", tp.Properties["hostIP"], "1.1.1.1", t)
	AssertEqual("podIP", tp.Properties["podIP"], "2.2.2.2", t)
	AssertEqual("restarts", tp.Properties["restarts"], uint(2), t)
	AssertEqual("startedAt", tp.Properties["startedAt"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("status", tp.Properties["status"], string(v1.PodRunning), t)

}
