package mqttinflux

import (
    "errors"
    "fmt"
    "log"
    "net/http"
    "regexp"
    "strings"
    "time"
)

var measurementPattern = regexp.MustCompile("^[a-zA-Z0-9-_\\.]+$")
var fieldPattern = regexp.MustCompile("^[a-zA-Z0-9-_\\.]+$")
var tagPattern = regexp.MustCompile("^[a-zA-Z0-9-_\\.]+$")

var influxQueue = make(chan *Measurement, 32)
var influxClient http.Client
var influxURL string
var influxUser string
var influxPass string

func startInflux(config Config) error {
    influxUser = config.InfluxUser
    influxPass = config.InfluxPass
    influxURL = fmt.Sprintf("http://%v:%v/write?db=%v",
        config.InfluxHost, config.InfluxPort, config.InfluxDB)

    go work()
    return nil
}

func stopInflux() {
    // TODO stop worker - opt: wait for complete
}

func submitMeasurement(m *Measurement) {
    influxQueue <- m
}

func send(m *Measurement) {
    err := m.Validate()
    if err != nil {
        log.Println(err)
        return
    }

    log.Printf("Influx send %v", m.Format())
    body := strings.NewReader(m.Format())
    req, err := http.NewRequest("POST", influxURL, body)
    if err != nil {
        log.Printf("ERROR: %v", err)
        return
    }
    req.SetBasicAuth(influxUser, influxPass)

    res, err := influxClient.Do(req)
    if err != nil {
        log.Printf("ERROR: %v", err)
        return
    }
    log.Println(res)
}

func work() {
    for {
        measurement, more := <-influxQueue
        if more {
            log.Println(measurement.Format())
            //send(measurement)
        }
    }
}

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
