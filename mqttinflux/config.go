package mqttinflux

import (
    "log"
)

type Config struct {
    MQTTHost string `json:"MQTTHost"`
    MQTTPort int `json:"MQTTPort"`
}



type Subscription struct {
    Topic string `json:"topic"`
}

func (s *Subscription) Handle(topic string, payload string) {
    log.Printf("Handle %v: %v", topic, payload)
}
