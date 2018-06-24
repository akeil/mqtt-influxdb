package mqttinflux

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

// AppName is the application name
const AppName = "mqtt-influxdb"

// set during build with -ldflags, see Makefile

// Version number.
var Version = ""

// Commit reference.
var Commit = ""

// Run starts the application.
// The `Run()` function will subscribe to all configured MQTT topics
// and wait for incoming messages until SIGINT is received.
func Run(configPath string) error {
	LogInfo("Starting %v Version %v (ref %v)", AppName, Version, Commit)

	// setup channel to receive SIGINT (ctrl+c) or SIGHUP (reload)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT)
	signal.Notify(s, syscall.SIGHUP)

	config, subscriptions, err := readSetup(configPath)
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

	err = start(config, subscriptions)
	if err != nil {
		return err
	}
	defer stop()

	// wait for SIGHUP or SIGINT
	for sig := range s {
		LogInfo("got signal %v", sig)
		if sig == syscall.SIGHUP {
			err = doReload(configPath)
			if err != nil {
				return err
			}
		} else {
			// assume SIGINT
			break
		}
	}
	return nil
}

// Reload configuration for another running instance of mqtt-influxdb.
//
// This is done by reading the pidfile (path taken from config)
// and sending a SIGHUP to that process.
func Reload(configPath string) error {
	config, err := readConfig(configPath)
	if err != nil {
		return err
	}

	pid, err := readPidFile(config.PidFile)
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	LogInfo("Sending SIGHUP to process %v", pid)
	return proc.Signal(syscall.SIGHUP)
}

// relaod configuration/subscriptions and re-subscribe to MQTT
func doReload(configPath string) error {
	LogInfo("reloading...")
	config, subscriptions, err := readSetup(configPath)
	if err != nil {
		return err
	}
	stop()
	return start(config, subscriptions)

}

func start(config Config, subscriptions []Subscription) error {
	err := connectMQTT(config, subscriptions)
	if err != nil {
		return err
	}
	err = startInflux(config)
	if err != nil {
		// redo the partial startup
		disconnectMQTT()
		return err
	}

	return nil
}

func stop() {
	disconnectMQTT()
	stopInflux()
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

	LogInfo("PID %v written to %q", pid, path)
	return nil
}

func removePidFile(path string) {
	LogInfo("remove PID file %q", path)
	os.Remove(path)
}

func readPidFile(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	raw := string(data)
	pid, err := strconv.ParseInt(raw, 10, 32)

	return int(pid), err
}

func readSetup(configPath string) (Config, []Subscription, error) {
	config, err := readConfig(configPath)
	if err != nil {
		return config, nil, err
	}

	subscriptions, err := loadSubscriptions()

	return config, subscriptions, err
}

func readConfig(configPath string) (Config, error) {
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

	var paths []string
	required := configPath != ""
	if configPath != "" {
		paths = []string{configPath}
	} else {
		currentUser, err := user.Current()
		if err != nil {
			return config, err
		}

		paths = []string{
			"/etc/" + AppName + ".json",
			filepath.Join(currentUser.HomeDir, ".config", AppName+".json"),
		}
	}

	found := false
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
		found = true

	}

	if required && !found {
		return config, errors.New("failed to read configuration")
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
		"/etc/" + AppName + ".d",
		filepath.Join(currentUser.HomeDir, ".config", AppName+".d"),
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
