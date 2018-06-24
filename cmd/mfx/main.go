package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/akeil/mqtt-influxdb/pkg"
)

func main() {
	var configPath string
	var printVersion bool
	var reload bool

	flag.StringVar(&configPath, "c", "",
		"Path, override default configuration file.")
	flag.BoolVar(&printVersion, "v", false,
		"Print version and exit.")
	flag.BoolVar(&reload, "r", false,
		"Reload configuration.")

	flag.Parse()

	if printVersion {
		fmt.Printf("%v %v (ref: %v)\n", mqttinflux.AppName,
			mqttinflux.Version, mqttinflux.Commit)
		return
	}

	var err error
	if reload {
		err = mqttinflux.Reload(configPath)
	} else {
		err = mqttinflux.Run(configPath)
	}

	if err != nil {
		log.Fatal(err)
	}
}
