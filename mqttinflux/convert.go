package mqttinflux

// format for field values by data type:
// - https://golang.org/pkg/fmt/
// - https://docs.influxdata.com/influxdb/v1.3//write_protocols/line_protocol_tutorial/#syntax
//
// float:
//    any numerical value, with or without decimal separator
//
// integer:
//    numerical value, no decimal separator, APPEND 'i' - e.g. 123i
//
//	  Go: %d, base 10 integer, add the 'i'
//
// boolean:
//    true: t, T, true, True, TRUE
//    false: f, F, false, False, FALSE
//
//	  Go: %t -> true|false
//
// string:
//    double quote, e.g. "foo" or "foo bar"
//    escape quotes within the string: "foo \"bar\" baz"
//
//    Go: %q, double quotes incl. escape

import (
	"fmt"
	"strconv"
	"strings"
)

// Converter is the type for a converter function.
type Converter func(raw string, params *Conversion) (string, error)

var converters map[string]Converter

func init() {
	converters = make(map[string]Converter, 3)
	converters["identity"] = Identity
	converters["float"] = Float
	converters["integer"] = Integer
	converters["string"] = String
	converters["boolean"] = Boolean
	converters["on-off"] = OnOff
}

// Conversion parameters
type Conversion struct {
	Kind      string            `json:"kind"`
	Precision int               `json:"precision"`
	Scale     float64           `json:"scale"`
	Lookup    map[string]string `json:"lookup"`
}

// Convert applies the conversion to the given `raw` string value.
func (c *Conversion) Convert(raw string) (string, error) {
	var err error

	if c.Lookup != nil {
		raw, err = c.translate(raw)
		if err != nil {
			return "", err
		}
	}

	key := c.Kind
	if key == "" {
		key = "identity"
	}
	conv, ok := converters[key]
	if !ok {
		return "", fmt.Errorf("conversion %q not supported", key)
	}
	return conv(raw, c)
}

// translate applies the Lookup map to the value
func (c *Conversion) translate(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	translated, found := c.Lookup[key]
	if !found {
		return "", fmt.Errorf("lookup failed for %q", key)
	}
	return translated, nil
}

// Identity is a `Convert` function which returns its input 1:1.
func Identity(raw string, params *Conversion) (string, error) {
	return raw, nil
}

// Float attempts to convert string input to a float value.
func Float(raw string, params *Conversion) (string, error) {
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "", err
	}

	// special case '-0.0' to '0.0'
	if parsed == -0 {
		parsed = 0
	}

	if params.Scale != 0 {
		parsed = parsed * params.Scale
	}

	template := "%f"
	if params.Precision != 0 {
		template = fmt.Sprintf("%%.%df", params.Precision)
	}
	return fmt.Sprintf(template, parsed), nil
}

// Integer converts input to a base 10 integer
func Integer(raw string, params *Conversion) (string, error) {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return "", err
	}

	if params.Scale != 0 {
		scaled := float64(parsed) * params.Scale
		parsed = int64(scaled)
	}

	return fmt.Sprintf("%di", parsed), nil
}

// String converts to a quoted string.
func String(raw string, params *Conversion) (string, error) {
	return fmt.Sprintf("%q", raw), nil
}

// Boolean converts to a boolean value.
func Boolean(raw string, params *Conversion) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	parsed, err := strconv.ParseBool(s)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%t", parsed), nil
}

// OnOff converts the string "on" or "off" to a boolean value
// on=true, off=false
// case-insensitive
func OnOff(raw string, params *Conversion) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	var value bool
	if s == "on" {
		value = true
	} else if s == "off" {
		value = false
	} else {
		return "", fmt.Errorf("expected on/off, got %q", raw)
	}
	return fmt.Sprintf("%t", value), nil
}
