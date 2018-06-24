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
	"strconv"
	"strings"
	"syscall"
)

// AppName is the application name
const AppName = "mqtt-influxdb"

// set during build with -ldflags, see Makefile

// Version number.
var Version = ""

// Commit reference.
var Commit = ""

var mqttService *MQTTService
var influxService *InfluxService

// Run starts the application.
// The `Run()` function will subscribe to all configured MQTT topics
// and wait for incoming messages until SIGINT is received.
func Run(configPath string) error {
	logStartup()

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

	// wait for SIGHUP (reload) or SIGINT (exit)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT)
	signal.Notify(s, syscall.SIGHUP)
	for sig := range s {
		logSignal(sig)
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

	logReadPID(config.PidFile)
	pid, err := readPidFile(config.PidFile)
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	logSendSignal(syscall.SIGHUP, pid)
	return proc.Signal(syscall.SIGHUP)
}

// reload configuration/subscriptions and re-subscribe to MQTT
func doReload(configPath string) error {
	LogInfo("reloading...")
	config, subs, err := readSetup(configPath)
	if err != nil {
		return err
	}
	stop()
	return start(config, subs)
}

func start(config Config, subs []Subscription) error {
	influxService = NewInfluxService(config)
	err := influxService.Start()
	if err != nil {
		return err
	}

	mqttService = NewMQTTService(config, influxService)
	mqttService.Register(subs)
	err = mqttService.Connect()
	if err != nil {
		// redo the partial startup
		influxService.Stop()
		return err
	}

	return nil
}

func stop() {
	mqttService.Disconnect()
	influxService.Stop()
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

	logPIDWritten(pid, path)
	return nil
}

func removePidFile(path string) {
	os.Remove(path)
	logPIDRemoved(path)
}

func readPidFile(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	raw := string(data)
	pid, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)

	return int(pid), err
}

func readSetup(configPath string) (Config, []Subscription, error) {
	config, err := readConfig(configPath)
	if err != nil {
		return config, nil, err
	}

	subs, err := readSubscriptions()

	return config, subs, err
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
	var required bool
	if configPath != "" {
		required = true
		paths = []string{configPath}
	} else {
		required = false
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
			logNoConfig(path)
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

	// having no config file is ok, *unless* one was explicitly specified
	if required && !found {
		return config, fmt.Errorf("failed to read configuration %q", configPath)
	}
	return config, nil
}

func readSubscriptions() ([]Subscription, error) {
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
			results, err := readSubscriptionFile(fullPath)
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

func readSubscriptionFile(path string) ([]Subscription, error) {
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

	logReadSubs(subs, path)
	return subs, nil
}

// Logging --------------------------------------------------------------------

func logStartup() {
	LogInfo("Controller starting %v - version %v (ref %v)", AppName, Version, Commit)
}

func logSignal(sig os.Signal) {
	LogInfo("Controller Received signal %v", sig)
}

func logReadSubs(subs []Subscription, path string) {
	LogInfo("Controller read %d subscriptions from '%v'", len(subs), path)
}

func logNoConfig(path string) {
	LogInfo("Controller no config found at '%v'", path)
}

func logPIDWritten(pid int, path string) {
	LogInfo("Controller PID %v written to %q", pid, path)
}

func logPIDRemoved(path string) {
	LogInfo("Controller removed PID file %q", path)
}

func logReadPID(path string) {
	LogInfo("Controller read PID from %q", path)
}

func logSendSignal(sig os.Signal, pid int) {
	LogInfo("Controller Sending signal %v to process %v", sig, pid)
}
