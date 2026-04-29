/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.

Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package reconciler

import (
	"reflect"
	"sync"
	"time"

	lru "github.com/golang/groupcache/lru"
	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/metrics"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/klog/v2"
)

// Size of the LRU cache used to find out of order delete/add sequences
const CACHE_SIZE = 500

// Public type for the complete state of the system.
// Looks a little different than the format of reconciler's internal state because this is friendlier
// for outside use by other packages
type CompleteState struct {
	Nodes                  []tr.Node // All the nodes
	Edges                  []tr.Edge // All the edges
	TotalNodes, TotalEdges int
}

// Public type for the diff state of the system since the previous.
// Looks a little different than the format of reconciler's internal state because this is friendlier
// for outside use by other packages
type Diff struct {
	AddNodes, UpdateNodes  []tr.Node     // Nodes to be added or updated
	DeleteNodes            []tr.Deletion // UIDs of nodes to be deleted
	AddEdges, DeleteEdges  []tr.Edge     // Edges to be added or deleted
	TotalNodes, TotalEdges int
}

// Create mapping with kind, namespace, and name as keys, and the Node itself as the value.
func nodeTripleMap(allNodes map[string]tr.Node) map[string]map[string]map[string]tr.Node {

	nodeMap := map[string]map[string]map[string]tr.Node{}
	for _, n := range allNodes {
		kind := n.Properties["kind"].(string) // blindly assert to string - it's always string
		namespace := ""
		if _, ok := n.Properties["namespace"]; !ok {
			namespace = "_NONE"
		} else {
			namespace = n.Properties["namespace"].(string)
		}
		// Initialize nodeMap for 'kind' if it doesn't exist already for that kind
		if _, ok := nodeMap[kind]; !ok {
			nodeMap[kind] = map[string]map[string]tr.Node{}
		}
		if _, ok := nodeMap[kind][namespace]; !ok {
			nodeMap[kind][namespace] = map[string]tr.Node{}
		}
		// Insert the name and uid mapping into nodeMap
		if name, ok := n.Properties["name"].(string); ok {
			nodeMap[kind][namespace][name] = n
		}
	}
	return nodeMap
}

// This object tracks and stores resources, and can regurgitate diffs based on the last time it was asked.
type Reconciler struct {
	currentNodes       map[string]tr.Node                         // Keyed by UID
	previousNodes      map[string]tr.Node                         // Keyed by UID
	diffNodes          map[string]tr.NodeEvent                    // Keyed by UID
	k8sEventNodes      map[string]tr.NodeEvent                    // Keyed by UID
	previousEventEdges map[string]tr.Edge                         // Keyed by UID
	edgeFuncs          map[string]func(ns tr.NodeStore) []tr.Edge // Edge building functions, keyed by UID

	previousEdges map[string]map[string]tr.Edge // Keyed by source then dest so we can quickly compare the new list
	totalEdges    int                           // Save the total count as we build to avoid looping when needed

	lastFullEdgeSync int64 // Unix timestamp of the last full allEdges() call within Diff()

	Input       chan tr.NodeEvent
	mutex       sync.Mutex // Used to protect currentState and diffState as they are accessed by multiple goroutines
	purgedNodes *lru.Cache // Tracks deleted nodes, so the reconciler can prevent out of order processing of events
}

// Creates a new Reconciler with a nil Input. To use it, set the Input and then start sending things through.
func NewReconciler() *Reconciler {
	r := &Reconciler{
		currentNodes:       make(map[string]tr.Node),
		previousNodes:      make(map[string]tr.Node),
		diffNodes:          make(map[string]tr.NodeEvent),
		k8sEventNodes:      make(map[string]tr.NodeEvent),
		previousEventEdges: make(map[string]tr.Edge),
		edgeFuncs:          make(map[string]func(ns tr.NodeStore) []tr.Edge),

		mutex:       sync.Mutex{},
		purgedNodes: lru.New(CACHE_SIZE),
	}

	go r.receive() // start it listening on input channel

	return r
}

// Returns the diff between the current and previous states, and resets the diff.
// TODO the latter half of this function got pretty messy, it could use a refactor/rewrite
func (r *Reconciler) Diff() Diff {
	klog.V(4).Info("Reconciler is calculating diff from previous state.")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ret := Diff{}

	if len(r.diffNodes) == 0 {
		klog.V(5).Info("Reconciler has no events since the last reconcile.")
		// allEdges() modifies r.totalEdges, so we have to set these fields both here and after allEdges() call
		ret.TotalNodes = len(r.currentNodes)
		ret.TotalEdges = r.totalEdges
		return ret
	}

	// Fill out nodes
	for _, ne := range r.diffNodes {
		switch ne.Operation {
		case tr.Create:
			ret.AddNodes = append(ret.AddNodes, ne.Node)
		case tr.Update:
			ret.UpdateNodes = append(ret.UpdateNodes, ne.Node)
		case tr.Delete:
			ret.DeleteNodes = append(ret.DeleteNodes, tr.Deletion{UID: ne.UID})
		}
	}

	// Hybrid edge computation: use incrementalEdges() every diff cycle for CPU efficiency,
	// and fall back to allEdges() periodically (EDGE_RESYNC_RATE_MS, default 60s) to catch
	// edges that the incremental approach misses — for example, a Service whose selector
	// matches a Pod whose labels were updated, where the Service is not in diffNodes.
	// Missed edges are corrected within one resync interval; only the delta is sent to the
	// indexer (no ClearAll / complete payload).
	var newEdges map[string]map[string]tr.Edge
	edgeResyncInterval := int64(config.Cfg.EdgeResyncRateMS / 1000)
	if time.Now().Unix()-r.lastFullEdgeSync >= edgeResyncInterval {
		klog.V(4).Info("Periodic full edge recomputation to catch any missed incremental edges.")
		newEdges = r.allEdges()
		r.lastFullEdgeSync = time.Now().Unix()
	} else {
		newEdges = r.incrementalEdges()
	}

	// Edges added: present in newEdges but absent from previousEdges.
	for srcUID, destMap := range newEdges {
		for destUID, newEdge := range destMap {
			if _, ok := r.previousEdges[srcUID][destUID]; !ok {
				ret.AddEdges = append(ret.AddEdges, newEdge)
			}
		}
	}

	// Edges deleted: present in previousEdges but absent from newEdges, and whose source/dest
	// node is not already in ret.DeleteNodes (the indexer removes edges automatically when a
	// node is deleted, so we only need to send explicit DeleteEdges for orphaned edges).
	for srcUID, destMap := range r.previousEdges {
		for destUID, oldEdge := range destMap {
			if _, ok := newEdges[srcUID][destUID]; ok {
				continue // edge still exists
			}
			srcDeleted, destDeleted := false, false
			for _, delNode := range ret.DeleteNodes {
				if srcUID == delNode.UID {
					srcDeleted = true
					break
				} else if destUID == delNode.UID {
					destDeleted = true
				}
			}
			if !srcDeleted && !destDeleted {
				ret.DeleteEdges = append(ret.DeleteEdges, oldEdge)
			}
		}
	}

	// We are now done with the old list of previousEdges.
	// Next time this is called we will want the edges we just calculated to be the previous.
	r.previousEdges = newEdges

	r.resetDiffs()

	ret.TotalNodes = len(r.currentNodes)
	ret.TotalEdges = r.totalEdges
	return ret
}

// Returns the complete current state and resets the diff
func (r *Reconciler) Complete() CompleteState {
	klog.V(3).Info("Reconciler is building the complete state.")
	r.mutex.Lock()
	defer r.mutex.Unlock()

	allNodes := make([]tr.Node, 0, len(r.currentNodes)) // We know the size ahead of time
	for _, n := range r.currentNodes {
		allNodes = append(allNodes, n)
	}

	ret := CompleteState{
		Nodes: allNodes,
	}

	newEdges := r.allEdges()

	// Coerce to array
	for _, destMap := range newEdges {
		for _, newEdge := range destMap {
			ret.Edges = append(ret.Edges, newEdge)
		}
	}

	// We are now done with the old list of previousEdges.
	// Next time this is called we will want the edges we just calculated to be the previous.
	r.previousEdges = newEdges

	r.resetDiffs()

	ret.TotalNodes = len(r.currentNodes)
	ret.TotalEdges = r.totalEdges
	return ret
}

// incrementalEdges builds an updated edge map by starting from previousEdges and recomputing
// edges only for nodes in diffNodes. This is O(|diffNodes|) instead of O(N) for a full
// recomputation, making it suitable for the Diff() hot path between periodic full resyncs.
//
// Trade-off: edges from unchanged nodes that reference a node whose properties changed
// (e.g. a Service whose selector now matches a Pod with updated labels) will not be updated
// until the next periodic allEdges() call (controlled by EDGE_RESYNC_RATE_MS, default 60s).
func (r *Reconciler) incrementalEdges() map[string]map[string]tr.Edge {
	klog.V(4).Info("Reconciler is updating edges incrementally for changed nodes.")

	ns := tr.NodeStore{
		ByUID:               r.currentNodes,
		ByKindNamespaceName: nodeTripleMap(r.currentNodes),
	}

	// Start with shallow references to every bucket in previousEdges. Buckets are deep-copied
	// the first time they are modified (copy-on-write), so r.previousEdges is never mutated.
	// This avoids the O(|currentEdges|) full deep-copy and only pays for the buckets touched
	// by diffNodes — typically O(|diffNodes|) in the common case.
	newEdges := make(map[string]map[string]tr.Edge, len(r.previousEdges))
	for srcUID, destMap := range r.previousEdges {
		newEdges[srcUID] = destMap // shallow reference
	}

	// Track which buckets have already been independently copied.
	cloned := make(map[string]bool, len(r.diffNodes))

	// ensureCloned deep-copies a bucket on its first write so we never modify a shared reference.
	ensureCloned := func(srcUID string) {
		if cloned[srcUID] {
			return
		}
		if existing, ok := newEdges[srcUID]; ok {
			copied := make(map[string]tr.Edge, len(existing))
			for k, v := range existing {
				copied[k] = v
			}
			newEdges[srcUID] = copied
		}
		cloned[srcUID] = true
	}

	// Apply Application-first ordering from diffNodes, matching allEdges() behavior.
	// This ensures _hostingApplication metadata is populated before Subscription edges are built
	// when both Application and Subscription nodes are updated in the same diff cycle.
	var appUIDs, otherUIDs []string
	for uid, ne := range r.diffNodes {
		if kind, _ := ne.Properties["kind"].(string); kind == "Application" { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
			appUIDs = append(appUIDs, uid)
		} else {
			otherUIDs = append(otherUIDs, uid)
		}
	}

	for _, uid := range append(appUIDs, otherUIDs...) {
		ne := r.diffNodes[uid]

		// Remove outgoing edges FROM this node; it will be recomputed below or is being deleted.
		delete(newEdges, uid)
		cloned[uid] = true // bucket is gone; any new writes will create a fresh map

		switch ne.Operation {
		case tr.Delete:
			// Remove all incoming edges TO this deleted node from every source bucket.
			for srcUID, destMap := range newEdges {
				if _, ok := destMap[uid]; !ok {
					continue
				}
				ensureCloned(srcUID)
				delete(newEdges[srcUID], uid)
				if len(newEdges[srcUID]) == 0 {
					delete(newEdges, srcUID)
					delete(cloned, srcUID)
				}
			}
		default: // Create or Update
			if edgeFunc, ok := r.edgeFuncs[uid]; ok {
				edges := edgeFunc(ns)
				edges = append(edges, tr.CommonEdges(uid, ns)...)
				for _, edge := range edges {
					if _, ok := newEdges[edge.SourceUID]; !ok {
						newEdges[edge.SourceUID] = make(map[string]tr.Edge)
						cloned[edge.SourceUID] = true
					} else {
						ensureCloned(edge.SourceUID)
					}
					newEdges[edge.SourceUID][edge.DestUID] = edge
				}
			}
		}
	}

	totalEdges := 0
	for _, destMap := range newEdges {
		totalEdges += len(destMap)
	}
	r.totalEdges = totalEdges

	return newEdges
}

// Builds all edges for all the nodes.
// Keyed by srcUID then destUID for fast comparison with previous.
// This function reads from the state, locking left up to caller (complete and diff methods)
func (r *Reconciler) allEdges() map[string]map[string]tr.Edge {
	klog.V(4).Info("Reconciler is rebuilding edges for all nodes.")
	ret := make(map[string]map[string]tr.Edge)

	ns := tr.NodeStore{
		ByUID:               r.currentNodes,
		ByKindNamespaceName: nodeTripleMap(r.currentNodes),
	}

	// After building the nodestore, get all the application UIDs in appUIDs and others in otherUIDs.
	// Process the application nodes first while building edges so that _hostingApplication metadata
	// gets populated for subscription nodes
	allUIDs := make([]string, len(r.edgeFuncs))

	// Copy all uids from reconciler edgeFuncs
	i := 0
	for uid := range r.edgeFuncs {
		allUIDs[i] = uid
		i++
	}
	// Filter all application nodes, store their UIDs in appUIDs
	apps := ns.ByKindNamespaceName["Application"]
	var appUIDs []string
	for namespace := range apps {
		for name := range apps[namespace] {
			appUIDs = append(appUIDs, apps[namespace][name].UID)
		}
	}
	// Store non-app UIDs in otherUIDs
	otherUIDs := tr.SliceDiff(allUIDs, appUIDs)

	// Loop across all the nodes and build their edges.
	for _, uid := range append(appUIDs, otherUIDs...) {
		klog.V(6).Infof("Calculating edges for node with UID: %s", uid)
		edges := r.edgeFuncs[uid](ns) // Get edges from this specific node

		edges = append(edges, tr.CommonEdges(uid, ns)...) // Get common edges for this node
		for _, edge := range edges {
			if _, ok := ret[edge.SourceUID]; !ok { // Init if it's not there
				ret[edge.SourceUID] = make(map[string]tr.Edge)
			}
			ret[edge.SourceUID][edge.DestUID] = edge
		}
	}

	totalEdges := 0
	// loop over double map to get the total number we added
	for _, destUID := range ret {
		totalEdges += len(destUID)
	}
	r.totalEdges = totalEdges

	return ret
}

// This method takes a channel and constantly receives from it, reconciling the input with whatever is currently stored
func (r *Reconciler) receive() {
	klog.Info("Reconciler receive routine started.")
	for {
		r.reconcileNode()
	}
}

// This is a separate function so we can defer the mutex unlock and guarantee the lock is lifted every iteration
func (r *Reconciler) reconcileNode() {
	ne := <-r.Input

	// Take care of diffState and currentState
	// Have to lock before the if statements, little awkward but if we made the decision to go ahead and edit
	// and then blocked, we could end up getting out of order
	r.mutex.Lock()
	defer r.mutex.Unlock()

	metrics.EventsReceivedCount.WithLabelValues(ne.Node.ResourceString).Inc() //nolint:staticcheck // "could remove embedded field 'Node' from selector"

	// Check whether we already have this node in our diff/purged state with a more up to date time.
	// If so, we ignore the version of it we're currently processing.
	otherNode, inDiff := r.diffNodes[ne.Node.UID]             //nolint:staticcheck // "could remove embedded field 'Node' from selector"
	nodeInterface, inPurged := r.purgedNodes.Get(ne.Node.UID) //nolint:staticcheck // "could remove embedded field 'Node' from selector"

	if inDiff && otherNode.Time > ne.Time {
		return
	}
	if inPurged {
		purgedNode, ok := nodeInterface.(tr.NodeEvent)
		// If the event is already present in purged list , check if the purge time is
		// equal or greater than the current time. Then we can skip processing the event
		if ok && purgedNode.Time >= ne.Time {
			return
		}
	}

	previousNode, inPrevious := r.previousNodes[ne.Node.UID] //nolint:staticcheck // "could remove embedded field 'Node' from selector"

	if ne.Operation == tr.Delete {
		delete(r.currentNodes, ne.UID) // Get rid of it from our currentState, if it was ever there.
		delete(r.edgeFuncs, ne.UID)
		r.purgedNodes.Add(ne.UID, ne) // Add this to the list of node purged resources

		if inPrevious {
			r.diffNodes[ne.UID] = ne // Since it was in the previous, we need to have a deletion diff.
			metrics.ResourcesSentToIndexerCount.WithLabelValues(ne.Node.ResourceString).Inc()
		} else {
			delete(r.diffNodes, ne.UID) // Otherwise no need to send a payload, just remove from local memory
		}
	} else { // This is either an update or create, which look very similar. TODO actually combine the two.
		ne.Operation = tr.Create
		if inPrevious { // If this was in the previous, our operation for diffs is update, not create
			ne.Operation = tr.Update

			// skip updates if new event is redundant to our previous state
			// (a property that we don't care about triggered an update)
			// For nodes that are not applications or subscriptions, We only care about the Properties,
			// the Metadata is only used to compute the edges and not sent with the node data.
			skip := reflect.DeepEqual(ne.Node.Properties, previousNode.Properties) //nolint:staticcheck // "could remove embedded field 'Node' from selector"

			kind := ne.Node.Properties["kind"] //nolint:staticcheck // "could remove embedded field 'Node' from selector"

			// If the node is an application or subscription, it might have changes to its metadata we
			// need to account for so don't skip updates on those
			if skip && (kind == "Application" || kind == "Subscription") {
				skip = false
			}

			// VAPBs specially need to update edges based on this piece of metadata
			if skip && kind == "ValidatingAdmissionPolicyBinding" {
				if ne.Node.Metadata["paramRef"] != previousNode.Metadata["paramRef"] { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					skip = false
				}
			}

			isPolicy := (kind == "CertificatePolicy" || kind == "ConfigurationPolicy" || kind == "OperatorPolicy")
			isConstraint := ne.Node.Properties["apigroup"] == "constraints.gatekeeper.sh" //nolint:staticcheck // "could remove embedded field 'Node' from selector"

			if skip && (isPolicy || isConstraint) {
				if !reflect.DeepEqual(ne.Node.Metadata["relObjs"], previousNode.Metadata["relObjs"]) { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					skip = false
				}
			}

			if skip {
				return
			}
		}
		// Each configmap for a helm release triggers a releases tranformation . If there are N configmaps
		// we are processing the same helm release N times. Since the order which the configmap gets this point
		// is not gauranteed , we are setting helm status which are old . Skipping if the current helm revison
		// is OLDER than one we already have.
		if ne.Node.ResourceString == "releases" { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
			// If node has already been sent, check the previous helm revision is latest and discard current one
			if inPrevious {
				if previousNode.Properties["revision"].(int64) > ne.Node.Properties["revision"].(int64) { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					klog.V(5).Infof("Skip %d for  release %s - previous is good",
						ne.Node.Properties["revision"], ne.Node.Properties["name"]) //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					return
				}
			}
			// If we have processed this release already (ready to send), check it's the latest and discard current one
			if nodeVal, ok := r.currentNodes[ne.UID]; ok {
				if nodeVal.Properties["revision"].(int64) > ne.Node.Properties["revision"].(int64) { //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					klog.V(5).Infof("Skip %d for  release %s - lower revision",
						ne.Node.Properties["revision"], ne.Node.Properties["name"]) //nolint:staticcheck // "could remove embedded field 'Node' from selector"
					return
				}
			}
		}
		// TODO: log the resources that surpass a specific threshold of EventsReceivedCount - ResourcesSentToIndexerCount
		metrics.ResourcesSentToIndexerCount.WithLabelValues(ne.Node.ResourceString).Inc()

		r.currentNodes[ne.UID] = ne.Node
		r.edgeFuncs[ne.UID] = ne.ComputeEdges
		r.diffNodes[ne.UID] = ne
	}
}

// Clears out diffState and copies currentState into previousState.
// (has to actually make a copy, maps are normally pass by reference)
// NOT THREADSAFE with anything that edits structures in s, locking left up to the caller.
func (r *Reconciler) resetDiffs() {
	// We have to reset the diff every time we try to prepare something to send,
	// so that it doesn't get out of sync with the complete/old.
	r.diffNodes = make(map[string]tr.NodeEvent)
	r.previousNodes = make(map[string]tr.Node, len(r.currentNodes))

	for uid, node := range r.currentNodes {
		r.previousNodes[uid] = node
	}
}
