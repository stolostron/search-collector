package mq

type MqMessage struct {
	UID        string                 `json:"uid,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Checksum   uint64                 `json:"checksum,omitempty"`
}

type Store struct {
	resources map[string]*MqMessage
}
