package mqttinflux

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttClient mqtt.Client
var mqttSubscriptions = make([]string, 0)

func connectMQTT(config Config) error {
	uri := fmt.Sprintf("tcp://%v:%v", config.MQTTHost, config.MQTTPort)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(uri)

	hostname, err := os.Hostname()
	if err == nil {
		opts.SetClientID("mqtt-influxdb-" + hostname)
		opts.SetCleanSession(true)
	}

	mqttClient = mqtt.NewClient(opts) // global

	logMQTTConnect(uri)
	t := mqttClient.Connect()
	t.Wait() // no timeout
	return t.Error()
}

func disconnectMQTT() {
	if mqttClient != nil {
		if mqttClient.IsConnected() {
			logMQTTDisconnect()
			mqttClient.Disconnect(250) // 250 millis cleanup time
		}
	}
}

func subscribeMQTT(subscriptions []Subscription) error {
	var err error
	qos := byte(0)
	for _, sub := range subscriptions {
		logMQTTSubscribe(sub.Topic)
		s := sub // local var for scope
		t := mqttClient.Subscribe(s.Topic, qos, func(c mqtt.Client, m mqtt.Message) {
			handlingError := s.Handle(m.Topic(), string(m.Payload()))
			if handlingError != nil {
				logMQTTHandlingError(m.Topic(), handlingError)
			}
		})
		t.Wait() // no timeout
		err = t.Error()
		if err != nil {
			return err
		}
		mqttSubscriptions = append(mqttSubscriptions, s.Topic)
	}
	return nil
}

func unsubscribeMQTT() {
	if mqttClient != nil {
		for _, topic := range mqttSubscriptions {
			logMQTTUnsubscribe(topic)
			mqttClient.Unsubscribe(topic)
		}
	}
}

// Logging --------------------------------------------------------------------

func logMQTTConnect(uri string) {
	LogInfo("MQTT connect to '%v'", uri)
}

func logMQTTDisconnect() {
	LogInfo("MQTT disconnect")
}

func logMQTTSubscribe(topic string) {
	LogInfo("MQTT subscribe to '%v'", topic)
}

func logMQTTUnsubscribe(topic string) {
	LogInfo("MQTT unsubscribe from '%v'", topic)
}

func logMQTTHandlingError(topic string, err error) {
	LogError("MQTT Failed to handle message '%v': %v", topic, err)
}
