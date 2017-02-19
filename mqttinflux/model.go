package mqttinflux

import (
    "bytes"
    "errors"
    "fmt"
    "log"
    "strings"
    "text/template"
    "time"
)

// Config ---------------------------------------------------------------------

type Config struct {
    MQTTHost string `json:"MQTTHost"`
    MQTTPort int `json:"MQTTPort"`
    InfluxHost string `json:"influxHost"`
    InfluxPort int `json:"influxPort"`
    InfluxUser string `json:"influxUser"`
    InfluxPass string `json:"influxPass"`
    InfluxDB string `json:"InfluxDB"`
}

// Subscription ---------------------------------------------------------------

type Subscription struct {
    Topic string `json:"topic"`
    Measurement string `json:"measurement"`
    Tags map[string]string `json:"tags"`
    Conversion Conversion `json:"conversion"`
    cachedTemplates map[string]*template.Template `json:"-"`
}

func (s *Subscription) parseTemplates() error {
    if s.cachedTemplates != nil {
        return nil
    }

    count := 1 + len(s.Tags)
    raw := make(map[string]string, count)
    s.cachedTemplates = make(map[string]*template.Template, count)
    raw["measurement"] = s.Measurement

    for k, v := range s.Tags {
        raw["tag." + k] = v
    }

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

    converted, err := s.Conversion.Convert(payload)
    if err != nil {
        return err
    }
    m.SetValue(converted)

    for tag, _ := range s.Tags {
        tagValue, err := s.fillTemplate("tag." + tag, ctx)
        if err != nil {
            return err
        }
        m.Tag(tag, tagValue)
    }

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

// Template -------------------------------------------------------------------

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

// Measurement ----------------------------------------------------------------

type Measurement struct {
    Name string
    Timestamp time.Time
    Values map[string]string
    Tags map[string]string
}

func NewMeasurement(name string) Measurement {
    m := Measurement{
        Name: name,
        Timestamp: time.Now(),
        Values: make(map[string]string, 0),
        Tags: make(map[string]string, 0),
    }
    return m
}

func (m *Measurement) Tag(name, value string) {
    m.Tags[name] = value
}

func (m *Measurement) SetValue(value string) {
    m.Values["value"] = value
}

func (m *Measurement) Format() string {
    // pattern:
    // <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
    // see:
    // https://docs.influxdata.com/influxdb/v1.2/write_protocols/line_protocol_reference/

    // <measurement>
    s := m.Name

    // ,<tag_key>=<tag_value>
    for tagname, tagvalue := range m.Tags {
        s += fmt.Sprintf(",%v=%v", tagname, tagvalue)
    }

    // <field_key>=<field_value>[,<field_key>=<field_value>]
    s += " "
    fieldCounter := 0
    fieldSeparator := ""
    for fieldName, fieldValue := range m.Values {
        if fieldCounter > 0 {
            fieldSeparator = ","
        }
        s += fmt.Sprintf("%v%v=%v", fieldSeparator, fieldName, fieldValue)
        fieldCounter++
    }

    //[ <timestamp>]
    s += fmt.Sprintf(" %d", m.Timestamp.UnixNano())
    return s
}

func (m *Measurement) Validate() error {
    if !measurementPattern.MatchString(m.Name) {
        return errors.New("Invalid measurement name")
    }

    if len(m.Values) == 0 {
        return errors.New("At least one value is required")
    }

    for fieldName, _ := range m.Values {
        if !fieldPattern.MatchString(fieldName) {
            return errors.New("Invalid field name")
        }
    }

    for tagName, _ := range m.Tags {
        if !tagPattern.MatchString(tagName) {
            return errors.New("Invalid tag name")
        }
    }

    return nil
}
