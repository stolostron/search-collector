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
		if r, ok := store.resources[res.Node.UID]; ok {
			// Skip if resource is in the store and has the same checksum hash.
			if r.Checksum == newChecksum {
				// klog.Infof("Skipping NodeEvent because checksum hash matches. %x\n", newHash)
				continue
			} else {
				// Resource is in the store but needs to be updated because checksum hash didn't match.
				newMqResource := MqMessage{
					UID:        res.Node.UID,
					Properties: res.Node.Properties,
					Checksum:   newChecksum,
				}

				klog.Infof("Sending UPDATE event to mq. Kind: %s\t Name: %s\n", newMqResource.Properties["kind"], newMqResource.Properties["name"])
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

				jsonBytes, jsonErr := json.Marshal(newMqResource)
				if jsonErr != nil {
					klog.Errorf("Error marshalling resource to json: %s", jsonErr)
					continue
				}
				if err := SendMessage(res.Node.UID, string(jsonBytes)); err == nil {

					// Update store
					store.resources[res.Node.UID] = &newMqResource
				} else {
					klog.Errorf("Error sending UPDATE event to mq: %s", err)
				}
			}
		} else {
			// Resource doesn't exist in store. Send message to mq and add to the store.
			newMqResource := MqMessage{
				UID:        res.Node.UID,
				Properties: res.Node.Properties,
				Checksum:   newChecksum,
			}
			klog.Infof("Sending CREATE event to mq. Kind: %s\t Name: %s\n", newMqResource.Properties["kind"], newMqResource.Properties["name"])

			jsonBytes, jsonErr := json.Marshal(newMqResource)
			if jsonErr != nil {
				klog.Errorf("Error marshalling resource to json: %s", jsonErr)
				continue
			}

			if err := SendMessage(res.Node.UID, string(jsonBytes)); err == nil {
				store.resources[res.Node.UID] = &newMqResource
			} else {
				klog.Errorf("Error sending CREATE event to mq: %s", err)
			}
		}

		// klog.Infof("Done processing NodeEvent.")
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
