package mqttinflux

import (
    "errors"
    "fmt"
    "strconv"
)

type Converter func(raw string, params *Conversion) (string, error)

var converters map[string]Converter

func init() {
    converters = make(map[string]Converter, 3)
    converters["identity"] = Identity
    converters["float"] = Float
    converters["integer"] = Integer
}

type Conversion struct {
    Kind string `json:"kind"`
    Precision int `json:"precision"`
}

func (c *Conversion) Convert(raw string) (string, error) {
    key := c.Kind
    if key == "" {
        key = "identity"
    }
    conv, ok := converters[key]
    if !ok {
        return "", errors.New("Conversion not supported")
    }
    return conv(raw, c)
}

func Identity(raw string, params *Conversion) (string, error) {
    return raw, nil
}

func Float(raw string, params *Conversion) (string, error) {
    parsed, err := strconv.ParseFloat(raw, 64)
    if err != nil {
        return "", err
    }

    template := "%f"
    if params.Precision != 0 {
        template = fmt.Sprintf(".%d%%f", params.Precision)
    }
    return fmt.Sprintf(template, parsed), nil
}

// base 10 integer
func Integer(raw string, params *Conversion) (string, error) {
    parsed, err := strconv.ParseInt(raw, 10, 64)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%d", parsed), nil
}

/*
func Round(raw string, params *Conversion) (string, error) {

}

func Ceil(raw string, params *Conversion) (string, error) {

}

func Floor(raw string, params *Conversion) (string, error) {

}
*/
