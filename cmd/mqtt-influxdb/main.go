package main


import (
    "log"

    "akeil.net/akeil/mqtt-influxdb/mqttinflux"
)


func main() {
    err := mqttinflux.Run()
    log.Println("hello")
    log.Println(err)
    if err != nil {
        log.Fatal(err)
    }
}
