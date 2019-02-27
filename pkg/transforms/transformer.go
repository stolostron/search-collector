package transforms

import (
	"errors"

	v1 "k8s.io/api/core/v1"                            // This one has all the concrete types
	machineryV1 "k8s.io/apimachinery/pkg/apis/meta/v1" // This one has the interface
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
	Uid        string                 `json: uid`
	Operation  Operation              `json: operation`
	Properties map[string]interface{} `json: properties`
}

// Object that handles transformation of k8s objects.
// To use, create one, call Start(), and begin passing in objects.
type Transformer struct {
	Input  chan machineryV1.Object // Put k8s objects into here.
	Output chan Node               // And recieve your redisgraph nodes from here.
	// TODO add stopper channel?
}

// Starts the transformer with a specified number of routines
func (t Transformer) Start(numRoutines int) error {
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
func transformRoutine(input chan machineryV1.Object, output chan Node) {
	// TODO not exactly sure, but we may need a stopper channel here.
	for {
		transformed := Node{}
		// Read from input channel
		resource := <-input

		// Type switch over input and call the appropriate transform function
		switch resource.(type) {
		case *v1.Pod:
			transformed = TransformPod(resource.(*v1.Pod))
		default:
			transformed = TransformCommon(resource)
		}

		// Send the result through the output channel
		output <- transformed
	}
}
