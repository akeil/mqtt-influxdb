package main

import (
	"log"

	"akeil.net/akeil/mqtt-influxdb/mqttinflux"
)

func main() {
	err := mqttinflux.Run()
	if err != nil {
		log.Fatal(err)
	}
}
