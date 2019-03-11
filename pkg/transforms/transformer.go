package transforms

import (
	"errors"
	"fmt"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchBeta "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"                          // This one has all the concrete types
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1" // This one has the interface
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
	Input        chan machineryV1.Object         // Put default k8s objects into here.
	DynamicInput chan *unstructured.Unstructured // Put nondefault k8s objects into here.
	Output       chan Node                       // And recieve your redisgraph nodes from here.
	// TODO add stopper channel?
}

// Starts the transformer with a specified number of routines
func (t Transformer) Start(numRoutines int) error {
	fmt.Println("Transformer started") // RM
	if numRoutines < 1 {
		return errors.New("numRoutines must be 1 or greater")
	}

	// start numRoutines threads to handle transformation.
	for i := 0; i < numRoutines; i++ {
		go transformRoutine(t.Input, t.DynamicInput, t.Output)
	}
	return nil
}

// This function is to be run as a goroutine that processes k8s objects into Nodes, then spits them out into the output channel.
func transformRoutine(input chan machineryV1.Object, dynamicInput chan *unstructured.Unstructured, output chan Node) {
	defer handleRoutineExit(input, dynamicInput, output)
	fmt.Println("Starting transformer routine")
	// TODO not exactly sure, but we may need a stopper channel here.
	for {
		var transformed Node

		// Read from one of the two input channels
		select {
		case resource := <-input: // Reading a default k8s object from the normal channel
			// Type switch over input and call the appropriate transform function
			switch typedResource := resource.(type) {
			case *core.ConfigMap:
				transformed = TransformConfigMap(typedResource)
			case *batchBeta.CronJob:
				transformed = TransformCronJob(typedResource)
			case *apps.DaemonSet:
				transformed = TransformDaemonSet(typedResource)
			case *apps.Deployment:
				transformed = TransformDeployment(typedResource)
			case *batch.Job:
				transformed = TransformJob(typedResource)
			case *core.Namespace:
				transformed = TransformNamespace(typedResource)
			case *core.Node:
				transformed = TransformNode(typedResource)
			case *core.PersistentVolume:
				transformed = TransformPersistentVolume(typedResource)
			case *core.Pod:
				transformed = TransformPod(typedResource)
			case *apps.ReplicaSet:
				transformed = TransformReplicaSet(typedResource)
			case *core.Secret:
				transformed = TransformSecret(typedResource)
			case *core.Service:
				transformed = TransformService(typedResource)
			case *apps.StatefulSet:
				transformed = TransformStatefulSet(typedResource)
			default:
				transformed = TransformCommon(typedResource)
			}
		case resource := <-dynamicInput: // Reading a nondefault object from the dynamic channel
			transformed = TransformUnstructured(resource)
		}

		// Send the result through the output channel
		output <- transformed
	}
}

// Handles a panic from inside transformRoutine.
// If the panic was due to an error, starts another transformRoutine with the same channels as this one.
// If not, just lets it die.
func handleRoutineExit(input chan machineryV1.Object, dynamicInput chan *unstructured.Unstructured, output chan Node) {
	// Recover and check the value. If we are here because of a panic, something will be in it.
	if r := recover(); r != nil { // Case where we got here from a panic
		fmt.Printf("Error in transformer routine: %v\n", r)

		// Start up a new routine with the same channels as the old one. The bad input will be gone since the old routine (the one that just crashed) took it out of the channel.
		go transformRoutine(input, dynamicInput, output)
	}
}
