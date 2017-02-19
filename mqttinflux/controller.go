package mqttinflux

import (
    "os"
    "os/signal"
)

func Run() error {
    var config Config
    // setup channel to receive SIGINT (ctrl+c)
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

    err := connectMQTT(config)
    if err != nil {
        return err
    }
    defer disconnectMQTT()

    err = subscribeMQTT(config)
    if err != nil {
        return err
    }
    defer unsubscribeMQTT()

    // wait for SIGINT
	_ = <-s
    return nil
}
