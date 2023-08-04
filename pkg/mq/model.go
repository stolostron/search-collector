package mq

type MqMessage struct {
	UID        string                 `json:"uid,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Hash       uint64                 `json:"hash,omitempty"`
}

type Store struct {
	resources map[string]*MqMessage
}
