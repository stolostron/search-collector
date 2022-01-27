/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/

package transforms

import (
	"runtime/debug"
	"strings"
	"sync"

	"github.com/golang/glog"
	appDeployable "github.com/stolostron/multicloud-operators-deployable/pkg/apis/apps/v1"
	rule "github.com/stolostron/multicloud-operators-placementrule/pkg/apis/apps/v1"
	acmapp "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
	appHelmRelease "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/helmrelease/v1"
	subscription "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/v1"
	application "sigs.k8s.io/application/api/v1beta1"

	policy "github.com/stolostron/governance-policy-propagator/pkg/apis/policies/v1"
	apps "k8s.io/api/apps/v1"

	batch "k8s.io/api/batch/v1"
	batchBeta "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/helm/pkg/proto/hapi/release"

	ocpapp "github.com/openshift/api/apps/v1"
)

// Operation is the event operation
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
	Metadata       map[string]string
}

func (n Node) hasMetadata(md string) bool {
	return n.Metadata != nil && n.Metadata[md] != ""
}

func (n Node) GetMetadata(md string) string {
	if n.hasMetadata(md) {
		return n.Metadata[md]
	}
	return ""
}

// These are the input to the sender. They have the node, and then they keep the time which is used for reconciling this version with other versions that the sender may already have.
type NodeEvent struct {
	Node
	ComputeEdges func(ns NodeStore) []Edge
	Time         int64
	Operation    Operation
}

type Deletion struct {
	UID string `json:"uid,omitempty"`
}

// make new constructor here, dry up code, then start testing

func NewNodeEvent(event *Event, trans Transform, resourceString string) NodeEvent {
	ne := NodeEvent{
		Time:         event.Time,
		Operation:    event.Operation,
		Node:         trans.BuildNode(),
		ComputeEdges: trans.BuildEdges,
	}
	ne.ResourceString = resourceString
	return ne
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
	SourceUID, DestUID   string
	SourceKind, DestKind string
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

var (
	NonNSResourceMap map[string]struct{} //store non-namespaced resources in this map
	NonNSResMapMutex = sync.RWMutex{}
)

func NewTransformer(inputChan chan *Event, outputChan chan NodeEvent, numRoutines int) Transformer {
	glog.Info("Transformer started")
	nr := numRoutines
	if numRoutines < 1 {
		glog.Warning(numRoutines, "is an invalid number of routines for Transformer. Using 1 instead.")
		nr = 1
	}

	// start numRoutines threads to handle transformation.
	for i := 0; i < nr; i++ {
		go TransformRoutine(inputChan, outputChan)
	}
	return Transformer{
		Input:  inputChan,
		Output: outputChan,
	}

}

// This function is to be run as a goroutine that processes k8s objects into Nodes, then spits them out into the output channel.
// If anything goes wrong in here that requires you to skip the current resource, call panic() and the routine will be spun back up by handleRoutineExit and the bad resource won't be in there because it was already taken out by the previous run.
func TransformRoutine(input chan *Event, output chan NodeEvent) {
	defer handleRoutineExit(input, output)
	glog.Info("Starting transformer routine")

	// TODO not exactly sure, but we may need a stopper channel here.
	for {
		var trans Transform

		event := <-input // Read from the input channel

		// Determine apiGroup and version of the resource
		apiGroup := ""

		if event.Resource.Object["apiVersion"] != nil && event.Resource.Object["apiVersion"] != "" {
			if apiVersionStr, ok := event.Resource.Object["apiVersion"].(string); ok {
				if len(strings.Split(apiVersionStr, "/")) == 2 {
					apiGroup = strings.Split(apiVersionStr, "/")[0]
				}
			}
		}
		kindApigroup := [2]string{event.Resource.GetKind(), apiGroup}
		//TODO: Might have to add more transform cases if resources like DaemonSet, StatefulSet etc. have other apigroups
		switch kindApigroup {
		case [2]string{"Application", "app.k8s.io"}:
			typedResource := application.Application{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ApplicationResourceBuilder(&typedResource)

		case [2]string{"Channel", "app.ibm.com"}, [2]string{"Channel", "apps.open-cluster-management.io"}:
			typedResource := acmapp.Channel{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ChannelResourceBuilder(&typedResource)

		case [2]string{"CronJob", "batch"}:
			typedResource := batchBeta.CronJob{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = CronJobResourceBuilder(&typedResource)

		case [2]string{"DaemonSet", "extensions"}:
			typedResource := apps.DaemonSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DaemonSetResourceBuilder(&typedResource)

		case [2]string{"DaemonSet", "apps"}:
			typedResource := apps.DaemonSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DaemonSetResourceBuilder(&typedResource)

		case [2]string{"Deployable", "app.ibm.com"}, [2]string{"Deployable", "apps.open-cluster-management.io"}:
			typedResource := appDeployable.Deployable{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = AppDeployableResourceBuilder(&typedResource)

		case [2]string{"Deployable", "policy.ibm.com"}:
			typedResource := appDeployable.Deployable{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeployableResourceBuilder(&typedResource)

		case [2]string{"Deployment", "apps"}:
			typedResource := apps.Deployment{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentResourceBuilder(&typedResource)

		case [2]string{"Deployment", "extensions"}:
			typedResource := apps.Deployment{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentResourceBuilder(&typedResource)

			//This is an ocp specific resource
		case [2]string{"DeploymentConfig", "apps.openshift.io"}:
			typedResource := ocpapp.DeploymentConfig{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentConfigResourceBuilder(&typedResource)

			//This is the application's HelmCR of kind HelmRelease.
		case [2]string{"HelmRelease", "apps.open-cluster-management.io"}:
			typedResource := appHelmRelease.HelmRelease{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = AppHelmCRResourceBuilder(&typedResource)

		case [2]string{"Job", "batch"}:
			typedResource := batch.Job{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = JobResourceBuilder(&typedResource)

		case [2]string{"Namespace", ""}:
			typedResource := core.Namespace{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NamespaceResourceBuilder(&typedResource)

		case [2]string{"Node", ""}:
			typedResource := core.Node{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NodeResourceBuilder(&typedResource)

		case [2]string{"PersistentVolume", ""}:
			typedResource := core.PersistentVolume{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PersistentVolumeResourceBuilder(&typedResource)

		case [2]string{"PersistentVolumeClaim", ""}:
			typedResource := core.PersistentVolumeClaim{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PersistentVolumeClaimResourceBuilder(&typedResource)

		case [2]string{"PlacementBinding", "policy.ibm.com"}, [2]string{"PlacementBinding", "apps.open-cluster-management.io"}:
			typedResource := policy.PlacementBinding{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementBindingResourceBuilder(&typedResource)

		case [2]string{"PlacementRule", "app.ibm.com"}, [2]string{"PlacementRule", "apps.open-cluster-management.io"}:
			typedResource := rule.PlacementRule{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementRuleResourceBuilder(&typedResource)

		case [2]string{"Pod", ""}:
			typedResource := core.Pod{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PodResourceBuilder(&typedResource)

		case [2]string{"Policy", "policy.open-cluster-management.io"}, [2]string{"Policy", "policies.open-cluster-management.io"}:
			typedResource := policy.Policy{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PolicyResourceBuilder(&typedResource)

		case [2]string{"ReplicaSet", "apps"}:
			typedResource := apps.ReplicaSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ReplicaSetResourceBuilder(&typedResource)

		case [2]string{"ReplicaSet", "extensions"}:
			typedResource := apps.ReplicaSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ReplicaSetResourceBuilder(&typedResource)

		case [2]string{"Service", ""}:
			typedResource := core.Service{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ServiceResourceBuilder(&typedResource)

		case [2]string{"StatefulSet", "apps"}:
			typedResource := apps.StatefulSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = StatefulSetResourceBuilder(&typedResource)

		case [2]string{"Subscription", "app.ibm.com"}, [2]string{"Subscription", "apps.open-cluster-management.io"}:
			typedResource := subscription.Subscription{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = SubscriptionResourceBuilder(&typedResource)

		default:
			trans = GenericResourceBuilder(event.Resource)
		}

		output <- NewNodeEvent(event, trans, event.ResourceString)

		if IsHelmRelease(event.Resource) {
			typedResource := core.ConfigMap{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(event.Resource.UnstructuredContent(), &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			releaseName := typedResource.GetLabels()["NAME"]
			release := getReleaseFromHelm(releaseName)
			if release == nil {
				AddToRetryChannel(event)
			}
			releaseTrans := HelmReleaseResource{&typedResource, release}
			output <- NewNodeEvent(event, releaseTrans, "releases")
		}
	}
}

//	If the resource is a ConfigMap with label "OWNER:TILLER", it references a Helm Release
func IsHelmRelease(resource *unstructured.Unstructured) bool {
	if kind := resource.GetKind(); kind == "ConfigMap" {
		for label, value := range resource.GetLabels() {
			if label == "OWNER" && value == "TILLER" {
				return true
			}
		}
	}
	return false
}

func getReleaseFromHelm(releaseName string) *release.Release {
	helmClient := GetHelmClient()
	if !HealthyConnection() {
		if len(helmReleaseRetry) < 1 {
			glog.Warning("Helm client not healthy; Cannot fetch helm release:", releaseName)
		}
		return nil
	}

	rc, err := helmClient.ReleaseContent(releaseName)
	if err != nil {
		glog.Warning("Failed to fetch helm release: ", releaseName, err)
		return nil
	}
	glog.V(3).Info("Retrieved helm release from Tiller: ", releaseName)
	return rc.GetRelease()
}

// Handles a panic from inside transformRoutine.
// If the panic was due to an error, starts another transformRoutine with the same channels as this one.
// If not, just lets it die.
func handleRoutineExit(input chan *Event, output chan NodeEvent) {
	// Recover and check the value. If we are here because of a panic, something will be in it.
	if r := recover(); r != nil { // Case where we got here from a panic
		glog.Errorf("Error in transformer routine: %v\n", r)
		glog.Error(string(debug.Stack()))

		// Start up a new routine with the same channels as the old one. The bad input will be gone since the old routine (the one that just crashed) took it out of the channel.
		go TransformRoutine(input, output)
	}
}
