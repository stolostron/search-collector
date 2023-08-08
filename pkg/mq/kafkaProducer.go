// Copyright Contributors to the Open Cluster Management project
package mq

import (
	"crypto/tls"

	"github.com/stolostron/search-collector/pkg/config"

	"github.com/IBM/sarama"
	"k8s.io/klog/v2"
)

var producer sarama.SyncProducer

func getProducerClient() (sarama.SyncProducer, error) {
	if producer != nil {
		return producer, nil
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Net.TLS.Enable = true
	saramaConfig.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal // This affects time to send message. Options: NoResponse, WaitForLocal, NoResponse
	saramaConfig.Producer.Retry.Max = config.Cfg.KafkaMaxRetry
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	var err error
	producer, err = sarama.NewSyncProducer(config.Cfg.KafkaBrokerList, saramaConfig)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return producer, nil
}

func SendMessage(uid string, msgJSON string) error {

	producer, err := getProducerClient()
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic:   config.Cfg.KafkaTopic,
		Value:   sarama.StringEncoder(msgJSON),
		Headers: []sarama.RecordHeader{sarama.RecordHeader{Key: []byte("clusterUID"), Value: []byte("TODO-CLUSTER-UID")}},
	}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return err
	}
	klog.Infof("Message is stored in topic(%s)/partition(%d)/offset(%d)\n", msg.Topic, partition, offset)

	return nil
}
