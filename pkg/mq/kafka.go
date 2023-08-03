package mq

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"k8s.io/klog/v2"
)

var (
	brokerList = []string{""}
	topic      = "cluster.A"
	maxRetry   = 3
	partition  = int32(0)
)

func SendMessage(uid string, msgJSON string) error {

	config := sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	config.Producer.RequiredAcks = sarama.WaitForAll // NOTE this affects time to send message. //sarama.NoResponse  //srarama.WaitForLocal
	config.Producer.Retry.Max = maxRetry
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		// log.Panic(err)
		return err
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Panic(err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msgJSON),
	}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		// log.Panic(err)
		return err
	}
	klog.Infof("Message is stored in topic(%s)/partition(%d)/offset(%d)\n", topic, partition, offset)

	return nil
}

func init() {
	initializeStoreFromMQ()
}

func initializeStoreFromMQ() {
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
			fmt.Println(err)
		case msg := <-consumer.Messages():
			readMsgCount++
			fmt.Printf("key: %s message: %s\n", string(msg.Key), string(msg.Value))

			// Parse message JSON

		}
	}

	klog.Info(">>> Done replaying existing messages from mq <<<\n")
}
