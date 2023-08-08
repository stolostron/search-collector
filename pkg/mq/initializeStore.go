// Copyright Contributors to the Open Cluster Management project
package mq

import (
	"crypto/tls"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
	"k8s.io/klog/v2"
)

var store Store

func initializeStoreFromKafka() {
	store = Store{
		resources: make(map[string]*MqMessage),
	}

	config := sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	main, err := sarama.NewConsumer(brokerList, config)
	if err != nil {
		log.Panic(err)
	}

	defer func() {
		if err := main.Close(); err != nil {
			log.Panic(err)
		}
	}()

	consumer, err := main.ConsumePartition(topic, partition, sarama.OffsetOldest)
	if err != nil {
		log.Panic(err)
	}

	klog.Info(">>> Read existing messages from mq to initialize state. <<<\n")

	client, clientErr := sarama.NewClient(brokerList, config)
	if clientErr != nil {
		log.Panic(clientErr)
	}

	offset, offsetErr := client.GetOffset(topic, partition, sarama.OffsetNewest)
	klog.Infof("Existing messages offset: %+v \toffsetErr:%+v\n", offset, offsetErr)

	for readMsgCount := 0; readMsgCount < int(offset); {
		select {
		case err := <-consumer.Errors():
			klog.Error(err)
		case msg := <-consumer.Messages():
			readMsgCount++
			// klog.Infof("key: %s message: %s\n", string(msg.Key), string(msg.Value))

			mqMessage := MqMessage{}
			jsonErr := json.Unmarshal(msg.Value, &mqMessage)
			if jsonErr != nil {
				klog.Errorf("Error unmarshalling mq message: %s", jsonErr)
				klog.Errorf("Skipping message: %+v", string(msg.Value))
				continue
			}

			store.resources[mqMessage.UID] = &mqMessage

		}
	}

	klog.Info(">>> Done replaying existing messages from mq and initializing local store.")
}
