/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"
	"time"

	"github.com/golang/glog"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func TestHelmTransformation(t *testing.T) {
	rel := []*release.Release{
		&release.Release{Name: "test"},
	}

	fc := &helm.FakeClient{
		Rels: rel,
	}

	// create a fast timer for fast test cases
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()
	output := make(chan NodeEvent)
	go HelmTransformation(fc, ticker.C, output)

	event := <-output
	if event.Operation != Create {
		glog.Fatal("Did not find create operation from helm node event")
	}
	if event.Properties["name"] != "test" {
		glog.Fatal("Couldn't not fetch example helm release")
	}

	// "delete" the chart by clearing the resource list
	fc.Rels = nil
	event = <-output

	if event.Operation != Delete {
		glog.Fatal("Did not find delete operation from helm node event")
	}
}
