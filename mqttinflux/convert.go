package mqttinflux

type Convert func(raw string) (string, error)

func Identity(raw string) (string, error) {
    return raw, nil
}
