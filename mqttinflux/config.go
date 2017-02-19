package mqttinflux

import (
    "log"
)

type Config struct {
    MQTTHost string `json:"MQTTHost"`
    MQTTPort int `json:"MQTTPort"`
    InfluxHost string `json:"influxHost"`
    InfluxPort int `json:"influxPort"`
    InfluxUser string `json:"influxUser"`
    InfluxPass string `json:"influxPass"`
    InfluxDB string `json:"InfluxDB"`
}

func readConfig() (Config, error) {
    // init with defaults
    config := Config{
        MQTTHost: "box",
        MQTTPort: 1883,
        InfluxHost: "box",
        InfluxPort: 8086,
        InfluxDB: "test",
        InfluxUser: "",
        InfluxPass: "",
    }

    // TODO: read from JSON file

    return config, nil
}

func loadSubscriptions() ([]Subscription, error) {
    subs := make([]Subscription, 0)

    s := Subscription{
        Topic: "test/foo",
    }
    subs = append(subs, s)

    return subs, nil
}

type Subscription struct {
    Topic string `json:"topic"`

}

func (s *Subscription) Handle(topic string, payload string) {
    log.Printf("Subscription: %v", s)
    log.Printf("Handle %v: %v", topic, payload)

    m := NewMeasurement(s.MeasurementName())
    m.SetValue(s.Value())

    // tags

    submitMeasurement(&m)
}

func (s *Subscription) MeasurementName() string {
    return "foo"
}

func (s *Subscription) Value() string {
    return "222"
}
