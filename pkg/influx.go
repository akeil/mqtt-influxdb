package mqttinflux

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var dbNamePattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
var measurementPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
var fieldPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")

//var valuePattern = regexp.MustCompile("^[a-zA-Z0-9:;\\-_\\.]*$")
var tagPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
var tagValuePattern = regexp.MustCompile("^[a-zA-Z0-9:;\\-_\\.]+$")

var influxQueue = make(chan *Measurement, 32)
var influxClient http.Client
var influxURL string
var influxDefaultDB string
var influxUser string
var influxPass string

func startInflux(config Config) error {
	influxUser = config.InfluxUser
	influxPass = config.InfluxPass
	influxDefaultDB = config.InfluxDB
	influxURL = fmt.Sprintf("http://%v:%v/write", config.InfluxHost,
		config.InfluxPort)

	logInfluxSettings(influxURL)

	go work()
	return nil
}

func stopInflux() {
	// TODO stop worker - opt: wait for complete
}

// submit a Measurement to the InfluxDB send queue.
// It will be sent asynchronously.
func submit(m *Measurement) {
	influxQueue <- m
}

func send(m *Measurement) error {
	err := m.Validate()
	if err != nil {
		return err
	}

	// DB name from measurement or default
	var dbName string
	if m.Database != "" {
		dbName = m.Database
	} else {
		dbName = influxDefaultDB
	}
	url := fmt.Sprintf("%v?db=%v", influxURL, dbName)

	body := strings.NewReader(m.Format() + "\n")
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(influxUser, influxPass)

	res, err := influxClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 || res.StatusCode == 204 {
		return nil
	}
	LogWarning("Got error for request (DB=%q): %q", dbName, m.Format())
	return fmt.Errorf("got HTTP %v", res.Status)
}

func work() {
	for {
		m, more := <-influxQueue
		if more {
			err := send(m)
			if err != nil {
				logInfluxSendError(err)
			}
		}
	}
}

// Logging --------------------------------------------------------------------

func logInfluxSettings(url string) {
	LogInfo("InfluxDB URL is '%v'", url)
}

func logInfluxSendError(err error) {
	LogError("InfluxDB failed to send measurement: %v", err)
}
