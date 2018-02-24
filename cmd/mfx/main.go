package main

import (
	"flag"
	"fmt"
	"log"

	"akeil.net/akeil/mqtt-influxdb/mqttinflux"
)

func main() {
	var configPath string
	var printVersion bool

	flag.StringVar(&configPath, "c", "",
		"Path, override default configuration file.")
	flag.BoolVar(&printVersion, "v", false,
		"Print version and exit.")

	flag.Parse()

	if printVersion {
		fmt.Printf("%v %v (ref: %v)\n", mqttinflux.APPNAME,
			mqttinflux.Version, mqttinflux.Commit)
		return
	}

	err := mqttinflux.Run(configPath)
	if err != nil {
		log.Fatal(err)
	}
}
