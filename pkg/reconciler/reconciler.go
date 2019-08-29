/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package reconciler

import (
	"reflect"
	"sync"

	"github.com/golang/glog"
	lru "github.com/golang/groupcache/lru"
	tr "github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

// Size of the LRU cache used to find out of order delete/add sequences
const CACHE_SIZE = 500

// Public type for the complete state of the system.
// Looks a little different than the format of reconciler's internal state because this is friendlier for outside use by other packages
type CompleteState struct {
	Nodes                  []tr.Node // All the nodes
	Edges                  []tr.Edge // All the edges
	TotalNodes, TotalEdges int
}

// Public type for the diff state of the system since the previous.
// Looks a little different than the format of reconciler's internal state because this is friendlier for outside use by other packages
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
	currentNodes  map[string]tr.Node      // Keyed by UID
	previousNodes map[string]tr.Node      // Keyed by UID
	diffNodes     map[string]tr.NodeEvent // Keyed by UID

	edgeFuncs map[string]func(ns tr.NodeStore) []tr.Edge // Edge building functions, keyed by UID

	previousEdges map[string]map[string]tr.Edge // Keyed by source then dest so we can quickly compare the new list
	totalEdges    int                           // Save the total count as we build to avoid looping when needed

	Input       chan tr.NodeEvent
	mutex       sync.Mutex // Used to protect currentState and diffState as they are edited and read by multiple goroutines
	purgedNodes *lru.Cache // Keep track of deleted nodes, so the reconciler can prevent out of order processing of events
}

// Creates a new Reconciler with a nil Input - you must set the Input and then start sending things through in order to use it.
func NewReconciler() *Reconciler {
	r := &Reconciler{
		currentNodes:  make(map[string]tr.Node),
		previousNodes: make(map[string]tr.Node),
		diffNodes:     make(map[string]tr.NodeEvent),

		edgeFuncs: make(map[string]func(ns tr.NodeStore) []tr.Edge),

		mutex:       sync.Mutex{},
		purgedNodes: lru.New(CACHE_SIZE),
	}

	go r.receive() // start it listening on input channel

	return r
}

// Returns the diff between the current and previous states, and resets the diff.
// TODO the latter half of this function got pretty messy, it could use a refactor/rewrite
func (r *Reconciler) Diff() Diff {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ret := Diff{}

	// Fill out nodes
	for _, ne := range r.diffNodes {
		if ne.Operation == tr.Create {
			ret.AddNodes = append(ret.AddNodes, ne.Node)
		} else if ne.Operation == tr.Update {
			ret.UpdateNodes = append(ret.UpdateNodes, ne.Node)
		} else if ne.Operation == tr.Delete {
			ret.DeleteNodes = append(ret.DeleteNodes, tr.Deletion{UID: ne.UID})
		}
	}

	// Fill out edges
	newEdges := r.allEdges()

	// TODO combine the following 2 loops?

	// Find elements that are in both new and old, and delete them from previous. After this, only the edges to be deleted will remain in previous.
	// TODO shortcut this by checking whether one of the src/dest doesn't exist any more (could delete the whole map in the case of srcUID missing)
	for srcUID, destMap := range newEdges {
		for destUID, newEdge := range destMap {
			if _, ok := r.previousEdges[srcUID][destUID]; ok { // If it's present in this loop it's obviously in the new set, so check the old
				delete(r.previousEdges[srcUID], destUID)
			} else { // If it's in the new and NOT the old, it's an edge that's been added
				ret.AddEdges = append(ret.AddEdges, newEdge)
			}
		}
	}

	// Now go back through the remains of the previous and coerce to slice of edges to be deleted
	for _, destMap := range r.previousEdges {
		for _, oldEdge := range destMap {
			ret.DeleteEdges = append(ret.DeleteEdges, oldEdge)
		}
	}

	// We are now done with the old list of previousEdges, next time this is called we will want the edges we just calculated to be the previous.
	r.previousEdges = newEdges

	r.resetDiffs()

	ret.TotalNodes = len(r.currentNodes)
	ret.TotalEdges = r.totalEdges
	return ret
}

// Returns the complete current state and resets the diff
func (r *Reconciler) Complete() CompleteState {
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

	// We are now done with the old list of previousEdges, next time this is called we will want the edges we just calculated to be the previous.
	r.previousEdges = newEdges

	r.resetDiffs()

	ret.TotalNodes = len(r.currentNodes)
	ret.TotalEdges = r.totalEdges
	return ret
}

// Builds all edges for all the nodes.
// Keyed by srcUID then destUID for fast comparison with previous.
// This function reads from the state, locking left up to caller (complete and diff methods)
func (r *Reconciler) allEdges() map[string]map[string]tr.Edge {
	ret := make(map[string]map[string]tr.Edge)

	ns := tr.NodeStore{
		ByUID:               r.currentNodes,
		ByKindNamespaceName: nodeTripleMap(r.currentNodes),
	}

	// Loop across all the nodes and build their edges.
	for uid, ef := range r.edgeFuncs {
		glog.V(3).Infof("Calculating edges UID: %s", uid)
		edges := ef(ns) // Get edges from this specific node

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

// This method takes a channel and constantly receives from it, reconciling the input with whatever is currently stored.
func (r *Reconciler) receive() {
	glog.Info("Reconciler Routine Started")
	for {
		r.reconcileNode()
	}
}

// This is a separate funcition so we can defer the mutex unlock and guarantee the lock is lifted every iteration
func (r *Reconciler) reconcileNode() {
	ne := <-r.Input

	// Take care of diffState and currentState
	r.mutex.Lock() // Have to lock before the if statements, little awkward but if we made the decision to go ahead and edit and then blocked, we could end up getting out of order
	defer r.mutex.Unlock()

	// Check whether we already have this node in our diff/purged state with a more up to date time. If so, we ignore the version of it we're currently processing.
	otherNode, inDiff := r.diffNodes[ne.Node.UID]
	nodeInterface, inPurged := r.purgedNodes.Get(ne.Node.UID)

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

	previousNode, inPrevious := r.previousNodes[ne.Node.UID]

	if ne.Operation == tr.Delete {
		delete(r.currentNodes, ne.UID) // Get rid of it from our currentState, if it was ever there.
		delete(r.edgeFuncs, ne.UID)
		r.purgedNodes.Add(ne.UID, ne) // Add this to the list of node purged resources

		if inPrevious {
			r.diffNodes[ne.UID] = ne // Since it was in the previous, we need to have a deletion diff.
		} else {
			delete(r.diffNodes, ne.UID) // Otherwise no need to send a payload, just remove from local memory
		}
	} else { // This is either an update or create, which look very similar. TODO actually combine the two.
		ne.Operation = tr.Create
		if inPrevious { // If this was in the previous, our operation for diffs is update, not create
			ne.Operation = tr.Update

			// skip updates if new event is redundant to our previous state (a property that we don't care about triggered an update)
			if reflect.DeepEqual(ne.Node, previousNode) {
				return
			}
		}
		r.currentNodes[ne.UID] = ne.Node
		r.edgeFuncs[ne.UID] = ne.ComputeEdges
		r.diffNodes[ne.UID] = ne
	}
}

// Clears out diffState and copies currentState into previousState. (has to actually make a copy, maps are normally pass by reference)
// NOT THREADSAFE with anything that edits structures in s, locking left up to the caller.
func (r *Reconciler) resetDiffs() {
	r.diffNodes = make(map[string]tr.NodeEvent) // We have to reset the diff every time we try to prepare something to send, so that it doesn't get out of sync with the complete/old.
	r.previousNodes = make(map[string]tr.Node, len(r.currentNodes))

	for uid, node := range r.currentNodes {
		r.previousNodes[uid] = node
	}
}
