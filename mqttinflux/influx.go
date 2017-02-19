package mqttinflux

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"
)

var influxQueue = make(chan *Measurement, 32)
var influxClient http.Client
var influxURL string
var influxUser string
var influxPass string

func startInflux(config Config) error {
    influxUser = config.InfluxUser
    influxPass = config.InfluxPass
    influxURL = fmt.Sprintf("http://%v:%v/write", config.InfluxHost, config.InfluxPort)

    go work()
    return nil
}

func stopInflux() {
    // TODO stop worker
}

func send(measurement *Measurement) {
    // pattern:
    // <measurement>,<tag>=<value>[,<tagN>=<valueN>] value=<value> <timestamp>
    s := ""
    body := strings.NewReader(s)  // io.Reader
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
            send(measurement)
        }
    }
}

type Measurement struct {
    Name string
    Value string
    Timestamp time.Time
}
