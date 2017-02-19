package mqttinflux

import (
    "fmt"
    "log"
    "net/http"
    "regexp"
    "strings"
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
