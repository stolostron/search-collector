package mq

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/klog/v2"
)

// Simple non-cryptographic hash function.
func hash(props map[string]interface{}) uint64 {
	algorithm := fnv.New64a()
	algorithm.Write([]byte(fmt.Sprintf("%s", props)))
	return algorithm.Sum64()
}

func MQReconciler(resources chan (tr.NodeEvent)) {
	for {
		res := <-resources

		newHash := hash(res.Node.Properties)

		// Check if resource is in store
		if r, ok := store.resources[res.Node.UID]; ok {
			// Skip if resource is in the store and has the same Hash.
			if r.Hash == newHash {
				klog.Infof("Skipping NodeEvent because Hash matches. %x\n", newHash)
				continue
			} else {
				// Resource is in the store but needs to be updated because Hash didn't match.
				newMqResource := MqMessage{
					UID:        res.Node.UID,
					Properties: res.Node.Properties,
					Hash:       newHash,
				}

				klog.Infof("Sending UPDATE message to mq. Kind: %s\t Name: %s\n", newMqResource.Properties["kind"], newMqResource.Properties["name"])
				// TODO: send only the changed bits to mq.

				jsonBytes, jsonErr := json.Marshal(newMqResource)
				if jsonErr != nil {
					klog.Errorf("Error marshalling resource to json: %s", jsonErr)
					continue
				}
				if err := SendMessage(res.Node.UID, string(jsonBytes)); err == nil {

					// Update store
					store.resources[res.Node.UID] = &newMqResource
				} else {
					klog.Errorf("Error sending UPDATE message to mq: %s", err)
				}
			}
		} else {
			// Resource doesn't exist in store. Send message to mq and add to the store.
			newMqResource := MqMessage{
				UID:        res.Node.UID,
				Properties: res.Node.Properties,
				Hash:       newHash,
			}
			klog.Infof("Sending CREATE to mq. Kind: %s\t Name: %s\n", newMqResource.Properties["kind"], newMqResource.Properties["name"])

			jsonBytes, jsonErr := json.Marshal(newMqResource)
			if jsonErr != nil {
				klog.Errorf("Error marshalling resource to json: %s", jsonErr)
				continue
			}

			if err := SendMessage(res.Node.UID, string(jsonBytes)); err == nil {
				store.resources[res.Node.UID] = &newMqResource
			} else {
				klog.Errorf("Error sending NEW message to mq: %s", err)
			}
		}

		// klog.Infof("Done processing NodeEvent.")
	}
}
