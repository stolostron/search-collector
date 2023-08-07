// Copyright Contributors to the Open Cluster Management project
package mq

import (
	"crypto/tls"
	"log"

	"github.com/IBM/sarama"
	"k8s.io/klog/v2"
)

func SendMessage(uid string, msgJSON string) error {
	config := sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	config.Producer.RequiredAcks = sarama.WaitForLocal // This affects time to send message. Options: NoResponse, WaitForLocal, NoResponse
	config.Producer.Retry.Max = maxRetry
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	// TODO: Use Async Producer.
	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		return err
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Panic(err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.StringEncoder(msgJSON),
		Headers: []sarama.RecordHeader{sarama.RecordHeader{Key: []byte("clusterUID"), Value: []byte("TODO-CLUSTER-UID")}},
	}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return err
	}
	klog.Infof("Message is stored in topic(%s)/partition(%d)/offset(%d)\n", topic, partition, offset)

	return nil
}
