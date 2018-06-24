package mqttinflux

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"
)

// Config settings.
type Config struct {
	PidFile    string `json:"pidfile"`
	MQTTHost   string `json:"MQTTHost"`
	MQTTPort   int    `json:"MQTTPort"`
	MQTTUser   string `json:"MQTTUser"`
	MQTTPass   string `json:"MQTTPass"`
	InfluxHost string `json:"influxHost"`
	InfluxPort int    `json:"influxPort"`
	InfluxUser string `json:"influxUser"`
	InfluxPass string `json:"influxPass"`
	InfluxDB   string `json:"influxDB"`
}

// Subscription describes a single subscription to an MQTT topic.
// the topic can contain wildcards.
//
// Topic: The MQTT topic to subscribe to
// Measurement: The InfluxDB measurement to wubmit to
// Database (optional): the name of the InfluxDB database. By default, the DB
//     from `Config` is used.
// Conversion: how to convert values from MQTT to InfluxDB.
type Subscription struct {
	Topic           string            `json:"topic"`
	Measurement     string            `json:"measurement"`
	Database        string            `json:"database"`
	Tags            map[string]string `json:"tags"`
	Conversion      Conversion        `json:"conversion"`
	cachedTemplates map[string]*template.Template
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
		raw["tag."+k] = v
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

// Handle an incoming message for the given topic.
func (s *Subscription) Handle(topic string, payload string) error {
	err := s.parseTemplates()
	if err != nil {
		return err
	}

	ctx := NewTemplateContext(topic, payload)
	measurementName, err := s.fillTemplate("measurement", ctx)
	if err != nil {
		return err
	}
	m := NewMeasurement(s.Database, measurementName)

	converted, err := s.Conversion.Convert(payload)
	if err != nil {
		return err
	}
	m.SetValue(converted)

	for tag := range s.Tags {
		tagValue, err := s.fillTemplate("tag."+tag, ctx)
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
		return "", fmt.Errorf("unknown template '%v'", name)
	}
	buf := new(bytes.Buffer)
	err := t.Execute(buf, &ctx)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Template -------------------------------------------------------------------

// A TemplateContext provides data for placeholders in templates.
type TemplateContext struct {
	Topic   string
	Payload string
	Parts   []string
}

// NewTemplateContext creates a new TemplateContext from an MQTT message.
func NewTemplateContext(topic, payload string) TemplateContext {
	return TemplateContext{
		Topic:   topic,
		Payload: payload,
		Parts:   strings.Split(topic, "/"),
	}
}

// Part returns a part from the MQTT topic.
func (ctx *TemplateContext) Part(index int) (string, error) {
	if index >= len(ctx.Parts) {
		return "", errors.New("Topic index out of range")
	}

	return ctx.Parts[index], nil
}

// JSON parses payload as JSON
// and gets a value from the resulting data structure
//
// `path` is a dotted path to access nested maps.
// e.g. `foo.bar.baz` would return `data['foo']['bar']['baz']`.
func (ctx *TemplateContext) JSON(path string) (string, error) {
	data := make(map[string]interface{})
	dec := json.NewDecoder(strings.NewReader(ctx.Payload))
	err := dec.Decode(&data)
	if err != nil {
		return "", err
	}

	parts := strings.Split(path, ".")
	context := data
	var current interface{}
	for index, key := range parts {
		current = context[key]
		switch current.(type) {
		case nil:
			return "", fmt.Errorf("could not find key '%v'", key)
		case map[string]interface{}:
			context = current.(map[string]interface{})
			// continue with the next path component
		default:
			if index != len(parts)-1 {
				return "", fmt.Errorf("could not find %v in JSON", path)
			}
			// we have reached the last path element, keep that value
			break
		}
	}

	// value to string
	return fmt.Sprintf("%v", current), nil
}

// Measurement is a single measurement to be submitted to InfluxDB.
type Measurement struct {
	Database  string
	Name      string
	Timestamp time.Time
	Values    map[string]string
	Tags      map[string]string
}

// NewMeasurement creates a new measurement for the given `database`
// and with the given `name`.
func NewMeasurement(database, name string) Measurement {
	m := Measurement{
		Database:  database,
		Name:      name,
		Timestamp: time.Now(),
		Values:    make(map[string]string, 0),
		Tags:      make(map[string]string, 0),
	}
	return m
}

// Tag sets a name/value pair as a tag on this measurement.
func (m *Measurement) Tag(name, value string) {
	m.Tags[name] = value
}

// SetValue sets the value for this measurement.
// The `value` is supplied in a string representation.
func (m *Measurement) SetValue(value string) {
	m.Values["value"] = value
}

// Format returns the "Line Protocol" representation for this measurement.
// See: https://docs.influxdata.com/influxdb/v1.5/write_protocols/line_protocol_reference/
func (m *Measurement) Format() string {
	// pattern:
	// <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]

	// <measurement>
	s := m.Name

	// sorted tags (for performance on recevier side)
	var tagNames []string
	for tagName := range m.Tags {
		tagNames = append(tagNames, tagName)
	}
	sort.Strings(tagNames)

	// ,<tag_key>=<tag_value>
	for _, tagName := range tagNames {
		tagValue := m.Tags[tagName]
		s += fmt.Sprintf(",%v=%v", tagName, tagValue)
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

// Validate this measurement.
func (m *Measurement) Validate() error {
	if m.Database != "" {
		if !dbNamePattern.MatchString(m.Database) {
			return errors.New("Invalid database name")
		}
	}

	if !measurementPattern.MatchString(m.Name) {
		return errors.New("Invalid measurement name")
	}

	if len(m.Values) == 0 {
		return errors.New("At least one value is required")
	}

	for fieldName := range m.Values {
		if !fieldPattern.MatchString(fieldName) {
			return errors.New("Invalid field name")
		}

		//if !valuePattern.MatchString(value) {
		//	return errors.New("Invalid value format")
		//}
	}

	for tagName, tagValue := range m.Tags {
		if !tagPattern.MatchString(tagName) {
			return errors.New("Invalid tag name")
		}

		if !tagValuePattern.MatchString(tagValue) {
			return errors.New("Invalid tag value")
		}
	}

	return nil
}
