package mqttinflux

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var measurementPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
var fieldPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
//var valuePattern = regexp.MustCompile("^[a-zA-Z0-9:;\\-_\\.]*$")
var tagPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
var tagValuePattern = regexp.MustCompile("^[a-zA-Z0-9:;\\-_\\.]+$")

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

	logInfluxSettings(influxURL)

	go work()
	return nil
}

func stopInflux() {
	// TODO stop worker - opt: wait for complete
}

func submitMeasurement(m *Measurement) {
	influxQueue <- m
}

func send(m *Measurement) error {
	err := m.Validate()
	if err != nil {
		return err
	}

	body := strings.NewReader(m.Format() + "\n")
	req, err := http.NewRequest("POST", influxURL, body)
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
	} else {
		LogWarning("Got error for request: %v", m.Format())
		return errors.New(fmt.Sprintf("Got HTTP %v", res.Status))
	}
}

func work() {
	for {
		measurement, more := <-influxQueue
		if more {
			err := send(measurement)
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
