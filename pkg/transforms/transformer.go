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
	Time           int64
	Operation      Operation
	Resource       *unstructured.Unstructured
	ResourceString string // This is a plural identifier of the kind, though in k8s this is called a "resource". e.g. for a pod, this is "pods"
}

// A generic node type that is passed to the aggregator for translation to whatever graphDB technology.
type Node struct {
	UID            string                 `json:"uid"`
	ResourceString string                 `json:"resourceString"`
	Properties     map[string]interface{} `json:"properties"`
}

// These are the input to the sender. They have the node, and then they keep the time which is used for reconciling this version with other versions that the sender may already have.
type NodeEvent struct {
	Node
	ComputeEdges func(ns NodeStore) []Edge
	Time         int64
	Operation    Operation
}

// A specific type designated for relationship type
type EdgeType string

// TODO: to be used later
// Differnet values for EdgeType
// const (
// 	ownedBy    EdgeType = "ownedBy"
// 	attachedTo EdgeType = "attachedTo"
// 	runsOn     EdgeType = "runsOn"
// 	selects    EdgeType = "selects"
// )

// Structure to hold Edge, containing the type and UIDs to relationships
type Edge struct {
	EdgeType
	SourceUID, DestUID string
}

// interface for each tranform
type Transform interface {
	BuildNode() Node
	BuildEdges(ns NodeStore) []Edge
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
		var trans Transform

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
			trans = ApplicationResource{&typedResource}

		case "ApplicationRelationship":
			typedResource := mcm.ApplicationRelationship{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ApplicationRelationshipResource{&typedResource}

		case "Compliance":
			typedResource := com.Compliance{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ComplianceResource{&typedResource}

		case "CronJob":
			typedResource := batchBeta.CronJob{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = CronJobResource{&typedResource}

		case "DaemonSet":
			typedResource := apps.DaemonSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DaemonSetResource{&typedResource}

		case "Deployable":
			typedResource := mcm.Deployable{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeployableResource{&typedResource}

		case "Deployment":
			typedResource := apps.Deployment{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentResource{&typedResource}

		case "Job":
			typedResource := batch.Job{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = JobResource{&typedResource}

		case "Namespace":
			typedResource := core.Namespace{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NamespaceResource{&typedResource}

		case "Node":
			typedResource := core.Node{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NodeResource{&typedResource}

		case "PersistentVolume":
			typedResource := core.PersistentVolume{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PersistentVolumeResource{&typedResource}

		case "PlacementBinding":
			typedResource := mcm.PlacementBinding{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementBindingResource{&typedResource}

		case "PlacementPolicy":
			typedResource := mcm.PlacementPolicy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementPolicyResource{&typedResource}

		case "Pod":
			typedResource := core.Pod{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PodResource{&typedResource}

		case "Policy":
			typedResource := policy.Policy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PolicyResource{&typedResource}

		case "ReplicaSet":
			typedResource := apps.ReplicaSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ReplicaSetResource{&typedResource}

		case "StatefulSet":
			typedResource := apps.StatefulSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = StatefulSetResource{&typedResource}
		default:
			trans = UnstructuredResource{event.Resource}
		}

		// Send the result through the output channel
		ne := NodeEvent{
			Time:         event.Time,
			Operation:    event.Operation,
			Node:         trans.BuildNode(),
			ComputeEdges: trans.BuildEdges,
		}
		ne.ResourceString = event.ResourceString
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
