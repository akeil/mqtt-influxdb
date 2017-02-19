package mqttinflux

import (
    "encoding/json"
    "log"
    "io"
    "os"
    "os/signal"
    "os/user"
    "path/filepath"
)

const APPNAME = "mqtt-influxdb"

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


func readConfig() (Config, error) {
    // init with defaults
    config := Config{
        MQTTHost: "localhost",
        MQTTPort: 1883,
        InfluxHost: "localhost",
        InfluxPort: 8086,
        InfluxDB: "default",
        InfluxUser: "",
        InfluxPass: "",
    }
    currentUser, err := user.Current()
	if err != nil {
		return config, err
	}

    paths := []string{
        "/etc/" + APPNAME + ".json",
        filepath.Join(currentUser.HomeDir, ".config", APPNAME + ".json"),
    }
    for _, path := range paths {
        f, err := os.Open(path)
        if os.IsNotExist(err){
            log.Printf("INFO: no config found at %v", path)
            continue
        } else if err != nil {
            return config, err
        }
        defer f.Close()

        decoder := json.NewDecoder(f)
        for {
            if err := decoder.Decode(&config); err == io.EOF {
                break
            } else if err != nil {
                return config, err
            }
        }

    }
    return config, nil
}
