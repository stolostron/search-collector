package mq

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/klog/v2"
)

func MQReconciler(resources chan (tr.NodeEvent)) {
	// Read previous state from kafka.
	initializeStoreFromKafka()

	// Process new data from informers, compare with previous state and send mq event if needed.
	for {
		res := <-resources

		newChecksum := checksum(res.Node.Properties)

		// Check if resource is in store
		lock.Lock()
		r, exist := store.resources[res.Node.UID]
		lock.Unlock()

		if exist {
			// Skip if resource is in the store and has the same checksum hash.
			if r.Checksum == newChecksum {
				continue
			} else {
				// Resource is in the store but needs to be updated because checksum hash didn't match.
				newMqResource := MqMessage{
					UID:        res.Node.UID,
					Properties: res.Node.Properties,
					Checksum:   newChecksum,
				}

				// klog.Infof("Sending UPDATE event to mq. Kind: %s\t Name: %s\n",
				// 	newMqResource.Properties["kind"], newMqResource.Properties["name"])

				// TODO: send only the changed bits to mq.
				// Data for delta message should look like this:
				// {
				//   properties: {
				// 	   newProp: "newValue",
				//     changedProp": "updatedValue",
				//	   deletedProp: null,
				//     listProp: [{remove: "value"}, {add: "value"}],
				//   }
				//   checksum: "The checksum hash of the entire object after updates."
				// }

				go sendMessageAndUpdateStore(res.Node.UID, &newMqResource)

			}
		} else {
			// Resource doesn't exist in store. Send message to mq and add to the store.
			newMqResource := MqMessage{
				UID:        res.Node.UID,
				Properties: res.Node.Properties,
				Checksum:   newChecksum,
			}
			// klog.Infof("Sending CREATE event to mq. Kind: %s\t Name: %s\n",
			// 	newMqResource.Properties["kind"], newMqResource.Properties["name"])

			go sendMessageAndUpdateStore(res.Node.UID, &newMqResource)

		}
	}
}

func sendMessageAndUpdateStore(uid string, mqResource *MqMessage) {
	jsonBytes, jsonErr := json.Marshal(mqResource)
	if jsonErr != nil {
		klog.Errorf("Error marshalling resource to json: %s", jsonErr)
		return
	}

	if err := SendMessage(uid, string(jsonBytes)); err == nil {
		// Update store
		lock.Lock()
		store.resources[uid] = mqResource
		lock.Unlock()
		klog.Infof("Done synchronizing. kind: %s\t name: %s\n", mqResource.Properties["kind"], mqResource.Properties["name"])
	} else {
		klog.Errorf("Error sending event to mq: %s", err)
	}
}

// Simple non-cryptographic hash function.
func checksum(props map[string]interface{}) uint64 {
	algorithm := fnv.New64a()
	_, err := algorithm.Write([]byte(fmt.Sprintf("%s", props)))
	if err != nil {
		klog.Errorf("Error calculating checksum: %s", err)
	}
	return algorithm.Sum64()
}
