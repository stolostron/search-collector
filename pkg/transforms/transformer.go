/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"encoding/json"
	"runtime/debug"
	"strings"
	"sync"

	mcmapp "github.com/IBM/multicloud-operators-channel/pkg/apis/app/v1alpha1"
	appDeployable "github.com/IBM/multicloud-operators-deployable/pkg/apis/app/v1alpha1"
	rule "github.com/IBM/multicloud-operators-placementrule/pkg/apis/app/v1alpha1"
	appHelmRelease "github.com/IBM/multicloud-operators-subscription-release/pkg/apis/app/v1alpha1"
	subscription "github.com/IBM/multicloud-operators-subscription/pkg/apis/app/v1alpha1"
	"github.com/golang/glog"

	// app "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"

	com "github.com/open-cluster-management/hcm-compliance/pkg/apis/compliance/v1alpha1"
	policy "github.com/open-cluster-management/hcm-compliance/pkg/apis/policy/v1alpha1"
	mcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"

	helmRelease "github.com/open-cluster-management/helm-crd/pkg/apis/helm.bitnami.com/v1"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchBeta "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/helm/pkg/proto/hapi/release"
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
		// Pull out the kind and use marshal/unmarshal to convert to the right type.
		j, err := json.Marshal(event.Resource)
		if err != nil {
			panic(err) // Will be caught by handleRoutineExit
		}

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
			typedResource := app.Application{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ApplicationResource{&typedResource}

		case [2]string{"ApplicationRelationship", "mcm.ibm.com"}:
			typedResource := mcm.ApplicationRelationship{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ApplicationRelationshipResource{&typedResource}

		case [2]string{"Channel", "app.ibm.com"}:
			typedResource := mcmapp.Channel{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ChannelResource{&typedResource}

		case [2]string{"Compliance", "compliance.mcm.ibm.com"}:
			typedResource := com.Compliance{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ComplianceResource{&typedResource}

		case [2]string{"CronJob", "batch"}:
			typedResource := batchBeta.CronJob{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = CronJobResource{&typedResource}

		case [2]string{"DaemonSet", "extensions"}:
			typedResource := apps.DaemonSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DaemonSetResource{&typedResource}

		case [2]string{"DaemonSet", "apps"}:
			typedResource := apps.DaemonSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DaemonSetResource{&typedResource}

		case [2]string{"Deployable", "app.ibm.com"}:
			typedResource := appDeployable.Deployable{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = AppDeployableResource{&typedResource}

		case [2]string{"Deployable", "mcm.ibm.com"}:
			typedResource := mcm.Deployable{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeployableResource{&typedResource}

		case [2]string{"Deployment", "apps"}:
			typedResource := apps.Deployment{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentResource{&typedResource}

		case [2]string{"Deployment", "extensions"}:
			typedResource := apps.Deployment{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = DeploymentResource{&typedResource}

			//This is the application's HelmCR of kind HelmRelease. From 2019 Q4, the apigroup will be app.ibm.com.
		case [2]string{"HelmRelease", "app.ibm.com"}:
			typedResource := appHelmRelease.HelmRelease{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = AppHelmCRResource{&typedResource}

		//This is the application's HelmCR of kind HelmRelease
		case [2]string{"HelmRelease", "helm.bitnami.com"}:
			typedResource := helmRelease.HelmRelease{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = HelmCRResource{&typedResource}

		case [2]string{"Job", "batch"}:
			typedResource := batch.Job{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = JobResource{&typedResource}

		case [2]string{"Namespace", ""}:
			typedResource := core.Namespace{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NamespaceResource{&typedResource}

		case [2]string{"Node", ""}:
			typedResource := core.Node{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = NodeResource{&typedResource}

		case [2]string{"PersistentVolume", ""}:
			typedResource := core.PersistentVolume{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PersistentVolumeResource{&typedResource}

		case [2]string{"PersistentVolumeClaim", ""}:
			typedResource := core.PersistentVolumeClaim{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PersistentVolumeClaimResource{&typedResource}

		case [2]string{"PlacementBinding", "mcm.ibm.com"}:
			typedResource := mcm.PlacementBinding{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementBindingResource{&typedResource}

		case [2]string{"PlacementPolicy", "mcm.ibm.com"}:
			typedResource := mcm.PlacementPolicy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementPolicyResource{&typedResource}

		case [2]string{"PlacementRule", "app.ibm.com"}:
			typedResource := rule.PlacementRule{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PlacementRuleResource{&typedResource}

		case [2]string{"Pod", ""}:
			typedResource := core.Pod{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PodResource{&typedResource}

		case [2]string{"Policy", "policy.mcm.ibm.com"}:
			typedResource := policy.Policy{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = PolicyResource{&typedResource}

		case [2]string{"ReplicaSet", "apps"}:
			typedResource := apps.ReplicaSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ReplicaSetResource{&typedResource}

		case [2]string{"ReplicaSet", "extensions"}:
			typedResource := apps.ReplicaSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ReplicaSetResource{&typedResource}

		case [2]string{"Service", ""}:
			typedResource := core.Service{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = ServiceResource{&typedResource}

		case [2]string{"StatefulSet", "apps"}:
			typedResource := apps.StatefulSet{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = StatefulSetResource{&typedResource}

		case [2]string{"Subscription", "app.ibm.com"}:
			typedResource := subscription.Subscription{}
			err = json.Unmarshal(j, &typedResource)
			if err != nil {
				panic(err) // Will be caught by handleRoutineExit
			}
			trans = SubscriptionResource{&typedResource}

		default:
			trans = UnstructuredResource{event.Resource}
		}

		output <- NewNodeEvent(event, trans, event.ResourceString)

		if IsHelmRelease(event.Resource) {
			typedResource := core.ConfigMap{}
			err = json.Unmarshal(j, &typedResource)
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
