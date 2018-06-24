package mqttinflux

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	dbNamePattern      = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
	measurementPattern = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
	fieldPattern       = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
	tagPattern         = regexp.MustCompile("^[a-zA-Z0-9\\-_\\.]+$")
	tagValuePattern    = regexp.MustCompile("^[a-zA-Z0-9:;\\-_\\.]+$")
)

// InfluxService represents an InfluxDB instance.
type InfluxService struct {
	queue     chan *Measurement
	client    *http.Client
	url       string
	defaultDB string
	user      string
	pass      string
}

// NewInfluxService creates a new InfluxService with the given config.
func NewInfluxService(config Config) *InfluxService {
	url := fmt.Sprintf("http://%v:%v/write", config.InfluxHost,
		config.InfluxPort)
	service := &InfluxService{
		queue:     make(chan *Measurement, 32),
		client:    &http.Client{},
		url:       url,
		user:      config.InfluxUser,
		pass:      config.InfluxPass,
		defaultDB: config.InfluxDB,
	}

	logInfluxSettings(url)

	return service
}

// Start sending measurements to the InfluxDB.
func (ifx *InfluxService) Start() error {
	go ifx.work()
	return nil
}

// Stop sending measurements to the InfluxDB.
func (ifx *InfluxService) Stop() {
	// TODO stop worker - opt: wait for complete
}

// Submit a Measurement to the InfluxDB send queue.
// It will be sent asynchronously.
func (ifx *InfluxService) Submit(m *Measurement) {
	ifx.queue <- m
}

func (ifx *InfluxService) work() {
	for {
		m, more := <-ifx.queue
		if more {
			err := ifx.send(m)
			if err != nil {
				logInfluxSendError(err)
			}
		}
	}
}

func (ifx *InfluxService) send(m *Measurement) error {
	err := m.Validate()
	if err != nil {
		return err
	}

	// DB name from measurement or default
	var dbName string
	if m.Database != "" {
		dbName = m.Database
	} else {
		dbName = ifx.defaultDB
	}
	url := fmt.Sprintf("%v?db=%v", ifx.url, dbName)

	body := strings.NewReader(m.Format() + "\n")
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(ifx.user, ifx.pass)

	res, err := ifx.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 || res.StatusCode == 204 {
		return nil
	}

	return fmt.Errorf("got HTTP status %v for DB=%q, req=&%q",
		res.Status, dbName, m.Format())
}

// Logging --------------------------------------------------------------------

func logInfluxSettings(url string) {
	LogInfo("InfluxDB URL is '%v'", url)
}

func logInfluxSendError(err error) {
	LogError("InfluxDB request error: %v", err)
}
