package mqttinflux

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTService manages subscriptions and the connection to the MQTT broker.
type MQTTService struct {
	uri    string
	subs   []Subscription
	client mqtt.Client
}

// NewMQTTService creates a new MQTTService based on the given `config`.
func NewMQTTService(config Config) *MQTTService {
	uri := fmt.Sprintf("tcp://%v:%v", config.MQTTHost, config.MQTTPort)
	service := &MQTTService{
		uri:  uri,
		subs: make([]Subscription, 0),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(uri)
	opts.SetUsername(config.MQTTUser)
	opts.SetPassword(config.MQTTPass)
	opts.OnConnect = service.OnConnect
	opts.OnConnectionLost = service.OnConnectionLost

	hostname, err := os.Hostname()
	if err == nil {
		opts.SetClientID("mqtt-influxdb-" + hostname)
		opts.SetCleanSession(true)
	}

	service.client = mqtt.NewClient(opts)

	return service
}

// Connect attempts to connect to the MQTT broker.
func (m *MQTTService) Connect() error {
	logMQTTConnecting(m.uri)
	t := m.client.Connect()
	t.Wait() // no timeout
	return t.Error()
}

// Disconnect closes the connection to the MQTT server
// and clears all subscribtions.
func (m *MQTTService) Disconnect() {
	if m.client.IsConnected() {
		logMQTTDisconnect()
		m.unsubscribe()
		m.client.Disconnect(250) // 250 millis cleanup time
	}
	m.clearSubscriptions()
}

// Subscribe to all registered subscribtions.
func (m *MQTTService) subscribe() error {
	var err error
	qos := byte(0)
	for _, sub := range m.subs {
		logMQTTSubscribe(sub.Topic)
		s := sub // local var for scope
		t := m.client.Subscribe(s.Topic, qos, func(c mqtt.Client, m mqtt.Message) {
			mmt, e := s.Read(m.Topic(), string(m.Payload()))
			if e != nil {
				logMQTTHandlingError(m.Topic(), e)
			}
			submit(&mmt)
		})
		t.Wait() // no timeout
		err = t.Error()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MQTTService) unsubscribe() {
	for _, sub := range m.subs {
		logMQTTUnsubscribe(sub.Topic)
		m.client.Unsubscribe(sub.Topic)
	}
}

// Register the given subscriptions. The MQTT service will subscribe to the
// respective topics as soon as it is connected to the broker.
func (m *MQTTService) Register(subs []Subscription) {
	// TODO: subscribe to broker *now* if we are connected
	for _, sub := range subs {
		m.subs = append(m.subs, sub)
	}
	logMQTTRegisteredSubscriptions()
}

func (m *MQTTService) clearSubscriptions() {
	m.subs = make([]Subscription, 0)
}

// OnConnect is the callback for an established connection.
func (m *MQTTService) OnConnect(c mqtt.Client) {
	opts := c.OptionsReader()
	logMQTTConnected(opts.Servers()[0].String())

	m.subscribe()
}

// OnConnectionLost is the callback for a lost connection.
func (m *MQTTService) OnConnectionLost(c mqtt.Client, reason error) {
	logMQTTConnectionLost(reason)
}

// Logging --------------------------------------------------------------------

func logMQTTConnecting(uri string) {
	LogInfo("MQTT connecting to '%v'", uri)
}

func logMQTTConnected(uri string) {
	LogInfo("MQTT (re-)connected to '%v'", uri)
}

func logMQTTDisconnect() {
	LogInfo("MQTT disconnecting")
}

func logMQTTConnectionLost(err error) {
	LogInfo("MQTT connection lost: '%v'", err)
}

func logMQTTSubscribe(topic string) {
	LogInfo("MQTT subscribe to '%v'", topic)
}

func logMQTTUnsubscribe(topic string) {
	LogInfo("MQTT unsubscribe from '%v'", topic)
}

func logMQTTRegisteredSubscriptions() {
	LogInfo("MQTT registered subscriptions")
}

func logMQTTHandlingError(topic string, err error) {
	LogError("MQTT Failed to handle message '%v': %v", topic, err)
}
