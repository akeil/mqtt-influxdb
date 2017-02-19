package mqttinflux

import (
    "fmt"
    "os"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttClient mqtt.Client
var mqttSubscriptions =make([]string, 0)

func connectMQTT(config Config) error {
    uri := fmt.Sprintf("tcp://%v:%v", config.MQTTHost, config.MQTTPort)
    opts := mqtt.NewClientOptions()
    opts.AddBroker(uri)

    hostname, err := os.Hostname()
    if err == nil {
        opts.SetClientID("mqtt-influxdb-" + hostname)
    }

    mqttClient = mqtt.NewClient(opts)  // global

    t := mqttClient.Connect()
    // block or timeout
    return t.Error()
}

func disconnectMQTT() {
    if mqttClient != nil {
        if mqttClient.IsConnected() {
            mqttClient.Disconnect(250)  // 250 millis cleanup time
        }
    }
}


func subscribeMQTT(config Config) error {
    qos := byte(0)
    subscriptions := make([]Subscription, 0)
    for _, sub := range subscriptions {
        s := sub  // local var for scope
        mqttClient.Subscribe(s.Topic, qos, func(c mqtt.Client, m mqtt.Message) {
            s.Handle(m.Topic(), string(m.Payload()))
        })
        mqttSubscriptions = append(mqttSubscriptions, s.Topic)
    }
    return nil
}

func unsubscribeMQTT() {
    if mqttClient != nil {
        for _, topic := range mqttSubscriptions {
            mqttClient.Unsubscribe(topic)
        }
    }
}
