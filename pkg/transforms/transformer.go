/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"encoding/json"

	"github.com/golang/glog"
	app "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
	mcm "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
	com "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/compliance/v1alpha1"
	policy "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchBeta "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Operation int

const (
	Create Operation = iota // 0
	Update                  // 1
	Delete                  // 2
)

// This type is used for add and update events.
type Event struct {
	Time      int64
	Operation Operation
	Resource  *unstructured.Unstructured
}

// A generic node type that is passed to the aggregator for translation to whatever graphDB technology.
type Node struct {
	UID        string                 `json:"uid"`
	Properties map[string]interface{} `json:"properties"`
}

// These are the input to the sender. They have the node, and then they keep the time which is used for reconciling this version with other versions that the sender may already have.
type NodeEvent struct {
	Node
	Time      int64
	Operation Operation
}

// Object that handles transformation of k8s objects.
// To use, create one, call Start(), and begin passing in objects.
type Transformer struct {
	Input  chan *Event    // Put your k8s resources and corresponding times in here.
	Output chan NodeEvent // And receive your aggregator-ready nodes (and times) from here.
	// TODO add stopper channel?
}

func NewTransformer(inputChan chan *Event, outputChan chan NodeEvent, numRoutines int) Transformer {
	glog.Info("Transformer started")
	nr := numRoutines
	if numRoutines < 1 {
		glog.Warning(numRoutines, "is an invalid number of routines for Transformer. Using 1 instead.")
		nr = 1
	}

	// start numRoutines threads to handle transformation.
	for i := 0; i < nr; i++ {
		go transformRoutine(inputChan, outputChan)
	}
	return Transformer{
		Input:  inputChan,
		Output: outputChan,
	}

}

// This function is to be run as a goroutine that processes k8s objects into Nodes, then spits them out into the output channel.
// If anything goes wrong in here that requires you to skip the current resource, call panic() and the routine will be spun back up by handleRoutineExit and the bad resource won't be in there because it was already taken out by the previous run.
func transformRoutine(input chan *Event, output chan NodeEvent) {
	defer handleRoutineExit(input, output)
	glog.Info("Starting transformer routine")
	// TODO not exactly sure, but we may need a stopper channel here.
	for {
		var transformed Node

		event := <-input // Read from the input channel
		// Pull out the kind and use marshal/unmarshal to convert to the right type.
		j, err := json.Marshal(event.Resource)
		if err != nil {
			panic(err) // Will be caught by handleRoutineExit
		}
		switch kind := event.Resource.GetKind(); kind {

		case "Application":
			typedResource := app.Application{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformApplication(&typedResource)

		case "ApplicationRelationship":
			typedResource := mcm.ApplicationRelationship{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformApplicationRelationship(&typedResource)

		case "Compliance":
			typedResource := com.Compliance{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformCompliance(&typedResource)

		case "ConfigMap":
			typedResource := core.ConfigMap{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformConfigMap(&typedResource)

		case "CronJob":
			typedResource := batchBeta.CronJob{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformCronJob(&typedResource)

		case "DaemonSet":
			typedResource := apps.DaemonSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformDaemonSet(&typedResource)

		case "Deployable":
			typedResource := mcm.Deployable{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformDeployable(&typedResource)

		case "DeployableOverride":
			typedResource := mcm.DeployableOverride{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformDeployableOverride(&typedResource)

		case "Deployment":
			typedResource := apps.Deployment{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformDeployment(&typedResource)

		case "Job":
			typedResource := batch.Job{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformJob(&typedResource)

		case "Namespace":
			typedResource := core.Namespace{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformNamespace(&typedResource)

		case "Node":
			typedResource := core.Node{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformNode(&typedResource)

		case "PersistentVolume":
			typedResource := core.PersistentVolume{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPersistentVolume(&typedResource)

		case "PlacementBinding":
			typedResource := mcm.PlacementBinding{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPlacementBinding(&typedResource)

		case "PlacementPolicy":
			typedResource := mcm.PlacementPolicy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPlacementPolicy(&typedResource)

		case "Pod":
			typedResource := core.Pod{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPod(&typedResource)

		case "Policy":
			typedResource := policy.Policy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPolicy(&typedResource)

		case "ReplicaSet":
			typedResource := apps.ReplicaSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformReplicaSet(&typedResource)

		case "Secret":
			typedResource := core.Secret{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformSecret(&typedResource)

		case "Service":
			typedResource := core.Service{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformService(&typedResource)

		case "StatefulSet":
			typedResource := apps.StatefulSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformStatefulSet(&typedResource)
		default:
			transformed = transformUnstructured(event.Resource)
		}

		// Send the result through the output channel
		ne := NodeEvent{
			Time:      event.Time,
			Operation: event.Operation,
			Node:      transformed,
		}
		output <- ne
	}
}

// Handles a panic from inside transformRoutine.
// If the panic was due to an error, starts another transformRoutine with the same channels as this one.
// If not, just lets it die.
func handleRoutineExit(input chan *Event, output chan NodeEvent) {
	// Recover and check the value. If we are here because of a panic, something will be in it.
	if r := recover(); r != nil { // Case where we got here from a panic
		glog.Errorf("Error in transformer routine: %v\n", r)

		// Start up a new routine with the same channels as the old one. The bad input will be gone since the old routine (the one that just crashed) took it out of the channel.
		go transformRoutine(input, output)
	}
}
