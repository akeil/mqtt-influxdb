package main

import (
	"flag"
	"log"

	"akeil.net/akeil/mqtt-influxdb/mqttinflux"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "",
		"Path, override default configuration file")
	flag.Parse()

	err := mqttinflux.Run(configPath)
	if err != nil {
		log.Fatal(err)
	}
}
