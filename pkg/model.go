package mqttinflux

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/jsonq"
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
// Value: optional, specify a template for the value
// Conversion: how to convert values from MQTT to InfluxDB.
type Subscription struct {
	Topic           string            `json:"topic"`
	Measurement     string            `json:"measurement"`
	Database        string            `json:"database"`
	Tags            map[string]string `json:"tags"`
	Value           string            `json:"value"`
	CSVSeparator    string            `json:"csvSeparator"`
	Conversion      Conversion        `json:"conversion"`
	cachedTemplates map[string]*template.Template
}

func (s *Subscription) parseTemplates() error {
	if s.cachedTemplates != nil {
		return nil
	}

	// measurement + value + tags
	count := 1 + 1 + len(s.Tags)
	raw := make(map[string]string, count)
	s.cachedTemplates = make(map[string]*template.Template, count)

	raw["measurement"] = s.Measurement
	raw["value"] = "{{." + s.Value + "}}"

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

// Read a Measurement from the given MQTT topic and payload.
func (s *Subscription) Read(topic, payload string) (Measurement, error) {
	var m Measurement
	err := s.parseTemplates()
	if err != nil {
		return m, err
	}

	ctx := NewTemplateContext(s, topic, payload)
	measurementName, err := s.fillTemplate("measurement", ctx)
	if err != nil {
		return m, err
	}
	m = NewMeasurement(s.Database, measurementName)

	// value from payload, optional template
	var rawValue string
	if s.Value == "" {
		rawValue = payload
	} else {
		rawValue, err = s.fillTemplate("value", ctx)
		if err != nil {
			return m, err
		}
	}

	converted, err := s.Conversion.Convert(rawValue)
	if err != nil {
		return m, err
	}
	m.SetValue(converted)

	for tag := range s.Tags {
		tagValue, err := s.fillTemplate("tag."+tag, ctx)
		if err != nil {
			return m, err
		}
		m.Tag(tag, tagValue)
	}

	return m, nil
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
	FullTopic    string
	Payload      string
	Parts        []string
	subscription *Subscription
}

// NewTemplateContext creates a new TemplateContext from an MQTT message.
func NewTemplateContext(subscription *Subscription, topic, payload string) TemplateContext {
	return TemplateContext{
		FullTopic:    topic,
		Payload:      payload,
		Parts:        strings.Split(topic, "/"),
		subscription: subscription,
	}
}

// Topic returns a part from the MQTT topic.
func (ctx *TemplateContext) Topic(index int) (string, error) {
	if index >= len(ctx.Parts) {
		return "", errors.New("Topic index out of range")
	}

	return ctx.Parts[index], nil
}

// JSON parses payload as JSON using jsonq
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

	query := jsonq.NewQuery(data)
	value, err := query.Interface(parts...)
	if err != nil {
		return "", err
	}

	// converts int, float, bool, etc to string
	return fmt.Sprintf("%v", value), nil
}

// CSV parses the payload as a CSV file and returns the value from `colIndex`
func (ctx *TemplateContext) CSV(colIndex int) (string, error) {
	payloadReader := strings.NewReader(ctx.Payload)
	csvReader := csv.NewReader(payloadReader)
	separator := ctx.subscription.CSVSeparator
	if separator != "" {
		runes := []rune(separator)
		if len(runes) != 1 {
			return "", fmt.Errorf("Invalid CSV separator %q", separator)
		}
		csvReader.Comma = runes[0]
	}

	records, err := csvReader.Read()
	if err != nil {
		return "", err
	}

	maxIndex := len(records) - 1
	if colIndex > maxIndex {
		return "", fmt.Errorf("column index %v is out of range (max: %v)",
			colIndex, maxIndex)
	}

	return records[colIndex], nil
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
