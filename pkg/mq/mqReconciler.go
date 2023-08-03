package mq

import (
	"crypto/sha256"
	"fmt"

	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/klog/v2"
)

type MqMessage struct {
	uid        string
	kindPlural string
	properties map[string]interface{}
	sha        string
}

type Store struct {
	resources map[string]*MqMessage
}

var store Store

func MQReconciler(resources chan (tr.NodeEvent)) {
	for {
		res := <-resources

		newShaBytes := sha256.Sum256([]byte(fmt.Sprintf("%s", res.Node.Properties)))
		newSha := fmt.Sprintf("%x", newShaBytes)
		klog.Infof("Procesing NodeEvent with sha: %x\n", newSha)

		// Check if resource is in store
		if r, ok := store.resources[res.Node.UID]; ok {
			// If it is, compare sha to check if it's the same.
			if r.sha == newSha {
				// If it is, skip
				klog.Infof("Skipping NodeEvent with sha: %x\n", newSha)
				continue
			} else {
				// If it's not, update store and send to mq
				newMqResource := MqMessage{
					uid:        res.Node.UID,
					kindPlural: res.Node.ResourceString,
					properties: res.Node.Properties,
					sha:        newSha,
				}

				klog.Infof("Sending UPDATE to mq:\n%+v\n", newMqResource)
				// TODO: send only the changed bits to mq.
				if err := SendMessage(res.Node.UID, fmt.Sprintf("%s", newMqResource)); err == nil { // TODO: send a json string

					// Update store
					store.resources[res.Node.UID] = &newMqResource
				} else {
					klog.Errorf("Error sending UPDATE message to mq: %s", err)
				}
			}
		} else {
			newMqResource := MqMessage{
				uid:        res.Node.UID,
				kindPlural: res.Node.ResourceString,
				properties: res.Node.Properties,
				sha:        newSha,
			}
			klog.Infof("Sending CREATE to mq:\n%+v\n", newMqResource)

			// send resource to mq.
			if err := SendMessage(res.Node.UID, fmt.Sprintf("%s", newMqResource)); err == nil { // TODO: send a json string

				// Add to store
				store.resources[res.Node.UID] = &newMqResource
			} else {
				klog.Errorf("Error sending NEW message to mq: %s", err)
			}
		}

		klog.Infof("Done processing NodeEvent.")
	}
}
