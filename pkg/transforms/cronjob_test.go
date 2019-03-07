package transforms

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	v1 "k8s.io/api/batch/v1beta1"
)

// Helper to create a resource for use in testing.
func CreateCronJob(t *testing.T) *v1.CronJob {
	var cj v1.CronJob

	rawCjBytes, err := ioutil.ReadFile("../../test-data/cronjob.json")
	if err != nil {
		t.Fatal("Unable to read test data", err)
	}
	err = json.Unmarshal(rawCjBytes, &cj) // Stuff the unmarshalled data into cj
	if err != nil {
		t.Fatal("Unable to unmarshal json to Cronjob", err)
	}

	return &cj
}

func TestTransformCronJob(t *testing.T) {
	res := CreateCronJob(t)

	tc := TransformCronJob(res)

	// Build time struct matching time in test data
	date := time.Date(2019, 3, 5, 23, 30, 0, 0, time.UTC)

	// Test only the fields that exist in cronjob - the common test will test the other bits
	AssertEqual("kind", tc.Properties["kind"], "CronJob", t)
	AssertEqual("active", tc.Properties["active"], 0, t)
	AssertEqual("lastSchedule", tc.Properties["lastSchedule"], date.UTC().Format(time.RFC3339), t)
	AssertEqual("schedule", tc.Properties["schedule"], "30 23 * * *", t)
	AssertEqual("suspend", tc.Properties["suspend"], false, t)
}
