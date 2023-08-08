// Copyright Contributors to the Open Cluster Management project
package mq

import (
	"crypto/tls"

	"github.com/stolostron/search-collector/pkg/config"

	"github.com/IBM/sarama"
	"k8s.io/klog/v2"
)

var producer sarama.SyncProducer

func SendMessage(uid string, msgJSON string) error {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Net.TLS.Enable = true
	saramaConfig.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal // This affects time to send message. Options: NoResponse, WaitForLocal, NoResponse
	saramaConfig.Producer.Retry.Max = config.Cfg.KafkaMaxRetry
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	if producer == nil {
		var err error
		producer, err = sarama.NewSyncProducer(config.Cfg.KafkaBrokerList, saramaConfig)
		if err != nil {
			return err
		}
	}

	// defer func() {
	// 	if err := producer.Close(); err != nil {
	// 		log.Panic(err)
	// 	}
	// }()

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
