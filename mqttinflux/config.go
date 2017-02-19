package mqttinflux

import (
    "bytes"
    "errors"
    "log"
    "strings"
    "text/template"
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
        Measurement: "test_{{.Part 1}}",
    }
    subs = append(subs, s)

    return subs, nil
}

type Subscription struct {
    Topic string `json:"topic"`
    Measurement string `json:"measurement"`

    cachedTemplates map[string]*template.Template `json:"-"`
}

func (s *Subscription) parseTemplates() error {
    if s.cachedTemplates != nil {
        return nil
    }

    count := 1
    raw := make(map[string]string, count)
    s.cachedTemplates = make(map[string]*template.Template, count)
    raw["measurement"] = s.Measurement

    for name, text := range raw {
        t := template.New(name)
        _, err := t.Parse(text)
        if err != nil {
            return err
        }
        s.cachedTemplates[name] = t
    }

    return nil
}

func (s *Subscription) Handle(topic string, payload string) error {
    log.Printf("Subscription: %v", s)
    log.Printf("Handle %v: %v", topic, payload)

    err :=s. parseTemplates()
    if err != nil {
        return err
    }

    ctx := NewTemplateContext(topic, payload)
    measurementName, err := s.fillTemplate("measurement", ctx)
    if err != nil {
        return err
    }
    m := NewMeasurement(measurementName)

    m.SetValue(payload)

    // tags

    submitMeasurement(&m)
    return nil
}

func (s *Subscription) fillTemplate(name string, ctx TemplateContext) (string, error) {
    t, ok := s.cachedTemplates[name]
    if !ok {
        return "", errors.New("unknown template")
    }
    buf := new(bytes.Buffer)
    err := t.Execute(buf, &ctx)
    if err != nil {
        return "", err
    }

    return buf.String(), nil
}

type TemplateContext struct {
    Topic string
    Payload string
    Parts []string
}

func NewTemplateContext(topic, payload string) TemplateContext {
    return TemplateContext{
        Topic: topic,
        Payload: payload,
        Parts: strings.Split(topic, "/"),
    }
}

func (ctx *TemplateContext) Part(index int) (string, error) {
    if index >= len(ctx.Parts) {
        return "", errors.New("Topic index out of range")
    }

    return ctx.Parts[index], nil
}
