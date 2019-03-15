package transforms

import (
	"encoding/json"
	"errors"

	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchBeta "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// used to track operations on Nodes/edges
type Operation string

const (
	Create Operation = "CREATE"
	Update Operation = "UPDATE"
	Delete Operation = "DELETE"
)

// A generic node type that is passed to the aggregator for translation to whatever graphDB technology.
type Node struct {
	UID        string                 `json:"uid"`
	Properties map[string]interface{} `json:"properties"`
}

// Object that handles transformation of k8s objects.
// To use, create one, call Start(), and begin passing in objects.
type Transformer struct {
	Input  chan *unstructured.Unstructured // Put your k8s resources in here.
	Output chan Node                       // And recieve your redisgraph nodes from here.
	// TODO add stopper channel?
}

// Starts the transformer with a specified number of routines
func (t Transformer) Start(numRoutines int) error {
	glog.Info("Transformer started") // RM
	if numRoutines < 1 {
		return errors.New("numRoutines must be 1 or greater")
	}

	// start numRoutines threads to handle transformation.
	for i := 0; i < numRoutines; i++ {
		go transformRoutine(t.Input, t.Output)
	}
	return nil
}

// This function is to be run as a goroutine that processes k8s objects into Nodes, then spits them out into the output channel.
// If anything goes wrong in here that requires you to skip the current resource, call panic() and the routine will be spun back up by handleRoutineExit and the bad resource won't be in there because it was already taken out by the previous run.
func transformRoutine(input chan *unstructured.Unstructured, output chan Node) {
	defer handleRoutineExit(input, output)
	glog.Info("Starting transformer routine")
	// TODO not exactly sure, but we may need a stopper channel here.
	for {
		var transformed Node

		resource := <-input // Read from the input channel
		// Pull out the kind and use marshal/unmarshal to convert to the right type.
		j, err := json.Marshal(resource)
		if err != nil {
			panic(err) // Will be caught by handleRoutineExit
		}
		switch kind := resource.GetKind(); kind {

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

		case "Pod":
			typedResource := core.Pod{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			transformed = transformPod(&typedResource)

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
			transformed = transformUnstructured(resource)
		}

		// Send the result through the output channel
		output <- transformed
	}
}

// Handles a panic from inside transformRoutine.
// If the panic was due to an error, starts another transformRoutine with the same channels as this one.
// If not, just lets it die.
func handleRoutineExit(input chan *unstructured.Unstructured, output chan Node) {
	// Recover and check the value. If we are here because of a panic, something will be in it.
	if r := recover(); r != nil { // Case where we got here from a panic
		glog.Errorf("Error in transformer routine: %v\n", r)

		// Start up a new routine with the same channels as the old one. The bad input will be gone since the old routine (the one that just crashed) took it out of the channel.
		go transformRoutine(input, output)
	}
}
