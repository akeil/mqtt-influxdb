package mqttinflux

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttClient mqtt.Client
var mqttSubscriptions = make([]Subscription, 0)

func connectMQTT(config Config, subscriptions []Subscription) error {
	for _, sub := range subscriptions {
		mqttSubscriptions = append(mqttSubscriptions, sub)
	}

	uri := fmt.Sprintf("tcp://%v:%v", config.MQTTHost, config.MQTTPort)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(uri)
	opts.OnConnect = connected
	opts.OnConnectionLost = connectionLost

	hostname, err := os.Hostname()
	if err == nil {
		opts.SetClientID("mqtt-influxdb-" + hostname)
		opts.SetCleanSession(true)
	}


	mqttClient = mqtt.NewClient(opts) // global

	logMQTTConnecting(uri)
	t := mqttClient.Connect()
	t.Wait() // no timeout
	return t.Error()
}

func disconnectMQTT() {
	if mqttClient != nil {
		if mqttClient.IsConnected() {
			logMQTTDisconnect()
			unsubscribe()
			mqttClient.Disconnect(250) // 250 millis cleanup time
		}
	}
}

func subscribe() error {
	var err error
	qos := byte(0)
	for _, sub := range mqttSubscriptions {
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
	}
	return nil
}

func unsubscribe() {
	if mqttClient != nil {
		for _, sub := range mqttSubscriptions {
			logMQTTUnsubscribe(sub.Topic)
			mqttClient.Unsubscribe(sub.Topic)
		}
	}
}

// Connection handlers --------------------------------------------------------

func connectionLost(client mqtt.Client, reason error) {
	logMQTTConnectionLost(reason)
}

func connected(client mqtt.Client) {
	opts := client.OptionsReader()
	logMQTTConnected(opts.Servers()[0])

	subscribeMQTT()
}

// Logging --------------------------------------------------------------------

func logMQTTConnecting(uri string) {
	LogInfo("MQTT connecting to '%v'", uri)
}

func logMQTTConnected(uri string) {
	LogInfo("MQTT (re-)connected to '%v'", uri)
}

func logMQTTDisconnect() {
	LogInfo("MQTT disconnecting")
}

func logMQTTConnectionLost(err error) {
	LogInfo("MQTT connection lost: '%v'", err)
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
