package mqttinflux

import (
//    "log"
    "os"
    "os/signal"
)

func Run() error {
    // setup channel to receive SIGINT (ctrl+c)
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

    config, err := readConfig()
    if err != nil {
        return err
    }

    subscriptions, err := loadSubscriptions()
    if err != nil {
        return err
    }

    err = startInflux(config)
    if err != nil {
        return err
    }
    defer stopInflux()

    err = connectMQTT(config)
    if err != nil {
        return err
    }
    defer disconnectMQTT()

    err = subscribeMQTT(subscriptions)
    if err != nil {
        return err
    }
    defer unsubscribeMQTT()

    // wait for SIGINT
	_ = <-s
    return nil
}
