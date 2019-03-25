package send

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

const (
	NEVER_SENT = "NEVER_SENT_HASH" // Constant used in place of the hash before the first time it ever sends.

)

// A UID of a node to be deleted, and the time at which it was deleted.
type DeleteNode struct {
	Time int64
	UID  string
}

// Represents the operations needed to bring the hub's version of this cluster's data into sync.
/* I'll put this back in once we add edges - Ethan
type Diff struct {
	nodes NodeDiff `json:"nodes"`
	// edges EdgeDiff
}
*/

// Represents the Node operations needed to bring the hub's version of this cluster's data into sync.
// FYI these are only exported because they need to be marhsalled and I didn't want to write my own Marshaller.
type NodeDiff struct {
	DeletedResources []string           `json:"deletedResources,omitempty"` // List of UIDs of nodes which need to be deleted
	AddResources     []*transforms.Node `json:"addedResources,omitempty"`   // List of Nodes that already existed which must be added
	UpdatedResources []*transforms.Node `json:"updatedResources,omitempty"` // List of Nodes that already existed which must be updated
}

// TODO We really should just import this from the aggregator so that they are guaranteed to match
// This type is for marshaling json which we send to the aggregator - it has to match the aggregator's API
type Payload struct {
	NodeDiff
	Hash     string `json:"hash,omitempty"`     // Hash of the previous state, used by aggregator to determine whether it needs to ask for the complete data
	ClearAll bool   `json:"clearAll,omitempty"` // Whether or not the aggregator should clear all data it has for the cluster first
}

// Keeps the total data for this cluster as well as the data since the last send operation.
type Sender struct {
	aggregatorURL        string // URL of the aggregator, minus any path
	aggregatorSyncPath   string // Path of the aggregator's POST route for syncing data, as of today /aggregator/clusters/{clustername}/sync
	aggregatorStatusPath string // Path of the aggregator's GET route for status, as of today /aggregator/clusters/{clustername}/status
	httpClient           http.Client
	totalState           []*transforms.Node // In the future this will be an object that has edges in it too TODO this should go to a map by UID so we can tell what is already there
	diffState            NodeDiff
	lastHash             string                     // The hash that was sent with the last send - the first step of a send operation is to ask the aggregator for this hash, to determine whether we can send a diff or need to send the complete data.
	lastSentTime         int64                      // Time at which we last sent data to the hub
	UpsertNodeChannel    chan transforms.UpsertNode // Put any nodes to be updated in here
	DeleteNodeChannel    chan *DeleteNode           // Put uids of any nodes to be deleted in here
	receiverMutex        sync.Mutex
}

// Constructs a new Sender using the provided channels.
func NewSender(upsertChan chan transforms.UpsertNode, deleteChan chan *DeleteNode, aggregatorURL, clusterName string) *Sender {

	// Construct senders
	s := &Sender{
		aggregatorURL:        aggregatorURL,
		aggregatorSyncPath:   strings.Join([]string{"/aggregator/clusters/", clusterName, "/sync"}, ""),
		aggregatorStatusPath: strings.Join([]string{"/aggregator/clusters/", clusterName, "/status"}, ""),
		httpClient:           http.Client{}, //default http client. TODO this will need credentials, ssl stuff, etc
		totalState:           make([]*transforms.Node, 0),
		diffState:            NodeDiff{},
		lastHash:             NEVER_SENT,
		lastSentTime:         -1,
		UpsertNodeChannel:    upsertChan,
		DeleteNodeChannel:    deleteChan,
	}

	// Attach functions to the channels
	go s.Receiver(upsertChan, "UPSERT")
	go DelReceiver(deleteChan, "DELETE")
	return s
}

/*
// Returns a payload containing the add, update and delete operations since the last send.
func (s *Sender) diffPayload() Payload {
	s.receiverMutex.Lock() // Can't be doing this when the receiver functions are changing the data, so lock them out.
	payload := Payload{
		NodeDiff: s.diffState,
		ClearAll: false,
		Hash:     s.lastHash,
	}
	s.receiverMutex.Unlock() // TODO should this be moved to outside this function?
	return payload
}
*/

// Returns a payload containing the complete set of resources as they currently exist in this cluster, at least up to the point that this sender knows about.
func (s *Sender) completePayload() Payload {
	s.receiverMutex.Lock() // Can't be doing this when the receiver functions are changing the data, so lock them out.
	payload := Payload{
		NodeDiff: NodeDiff{
			AddResources: s.totalState,
		},
		ClearAll: false, //FIXME This needs to be true, it is currently false to deal with temporary bug on the aggregator side
		// In this case, hash, update and delete aren't needed
	}
	s.receiverMutex.Unlock() // TODO should this be moved to outside this function?
	return payload
}

// First does a GET to the aggregator to see if it needs to send the entire data, or just a diff.
// Then, sends the data.
func (s *Sender) Send() error {
	// TODO HTTP GET and the diff case
	payload := s.completePayload()
	glog.Warningf("PAYLOAD: %+v", payload)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	glog.Warning(string(payloadBytes))
	payloadBuffer := bytes.NewBuffer(payloadBytes)
	_, err = s.httpClient.Post(s.aggregatorURL+s.aggregatorSyncPath, "application/json", payloadBuffer)
	if err != nil {
		return err
	}
	return nil
}

// Basic receiver that prints stuff, for my testing
func (s *Sender) Receiver(transformOutput chan transforms.UpsertNode, op string) {
	glog.Info(op, " Receiver started") //RM
	for {
		un := <-transformOutput
		n := un.Node
		name, ok := n.Properties["name"].(string)
		if !ok {
			name = "UNKNOWN"
		}

		kind, ok := n.Properties["kind"].(string)
		if !ok {
			kind = "UNKNOWN"
		}

		glog.Info(op, ": "+kind+" "+name)
		if un.Operation == transforms.Create {
			s.receiverMutex.Lock() // Don't want to edit while we read
			s.totalState = append(s.totalState, &un.Node)
			s.receiverMutex.Unlock()
		}
	}
}

// Basic receiver that prints stuff, for my testing
func DelReceiver(transformOutput chan *DeleteNode, s string) {
	glog.Info(s, " Receiver started") //RM
	for {
		dn := <-transformOutput

		glog.Info(s + ": UID " + dn.UID)

	}

}
