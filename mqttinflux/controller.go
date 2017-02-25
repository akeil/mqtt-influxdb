package mqttinflux

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

	if config.PidFile != "" {
		err = writePidFile(config.PidFile)
		if err != nil {
			return err
		}
		defer removePidFile(config.PidFile)
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

func writePidFile(path string) error {
	pid := os.Getpid()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d", pid))
	if err != nil {
		return err
	}

	return nil
}

func removePidFile(path string) {
	os.Remove(path)
}

func readConfig() (Config, error) {
	// init with defaults
	config := Config{
		PidFile:    "",
		MQTTHost:   "localhost",
		MQTTPort:   1883,
		InfluxHost: "localhost",
		InfluxPort: 8086,
		InfluxDB:   "default",
		InfluxUser: "",
		InfluxPass: "",
	}
	currentUser, err := user.Current()
	if err != nil {
		return config, err
	}

	paths := []string{
		"/etc/" + APPNAME + ".json",
		filepath.Join(currentUser.HomeDir, ".config", APPNAME+".json"),
	}
	for _, path := range paths {
		f, err := os.Open(path)
		if os.IsNotExist(err) {
			LogInfo("No config found at '%v'", path)
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

func loadSubscriptions() ([]Subscription, error) {
	subs := make([]Subscription, 0)

	currentUser, err := user.Current()
	if err != nil {
		return subs, err
	}
	dirnames := []string{
		"/etc/" + APPNAME + ".d",
		filepath.Join(currentUser.HomeDir, ".config", APPNAME+".d"),
	}

	for _, dirname := range dirnames {
		files, err := ioutil.ReadDir(dirname)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return subs, err
		}
		for _, file := range files {
			fullPath := filepath.Join(dirname, file.Name())
			results, err := loadSubscriptionFile(fullPath)
			if err != nil {
				return subs, err
			}
			for _, s := range results {
				subs = append(subs, s)
			}
		}
	}

	return subs, nil
}

func loadSubscriptionFile(path string) ([]Subscription, error) {
	subs := make([]Subscription, 0)

	f, err := os.Open(path)
	if err != nil {
		return subs, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for {
		if err := decoder.Decode(&subs); err == io.EOF {
			break
		} else if err != nil {
			return subs, err
		}
	}

	LogInfo("Loaded %d subscriptions from '%v'", len(subs), path)
	return subs, nil
}
