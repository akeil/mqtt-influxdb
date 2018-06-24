package mqttinflux

import (
	"testing"
)

func TestConvertIdentity(t *testing.T) {
	c := new(Conversion)

	raw := "foo"
	conv, err := c.Convert(raw)
	if err != nil {
		t.Error(err)
	} else if raw != conv {
		t.Fail()
	}
}

func TestConvertFloat(t *testing.T) {
	c := Conversion{Kind: "float", Precision: 2}
	cases := make(map[string]string, 10)
	cases["1"] = "1.00"
	cases["0"] = "0.00"
	cases["-1"] = "-1.00"
	cases["-0"] = "0.00"
	cases["100"] = "100.00"
	cases["1.1"] = "1.10"
	cases["2.123"] = "2.12"
	cases["2.789"] = "2.79"
	cases["03"] = "3.00"
	cases["03.04"] = "3.04"

	checkConversion(c, cases, t)

	expectedErrors := make([]string, 5)
	expectedErrors[0] = ""
	expectedErrors[1] = "foo"
	expectedErrors[2] = "0x01"
	expectedErrors[3] = "#01"
	expectedErrors[4] = "1,123"

	checkExpectedErrors(c, expectedErrors, t)
}

func TestConvertInteger(t *testing.T) {
	c := Conversion{Kind: "integer"}

	cases := make(map[string]string, 7)
	cases["1"] = "1i"
	cases["-1"] = "-1i"
	cases["0"] = "0i"
	cases["-0"] = "0i"
	cases["00"] = "0i"
	cases["01"] = "1i"
	cases["123"] = "123i"
	checkConversion(c, cases, t)

	expectedErrors := make([]string, 6)
	expectedErrors[0] = ""
	expectedErrors[1] = "foo"
	expectedErrors[2] = "0x01"
	expectedErrors[3] = "#01"
	expectedErrors[4] = "1.123"
	expectedErrors[5] = "1,123"

	checkExpectedErrors(c, expectedErrors, t)
}

func TestConvertString(t *testing.T) {
	c := Conversion{Kind: "string"}

	cases := make(map[string]string, 4)
	cases["foo"] = "\"foo\""
	cases["foo bar"] = "\"foo bar\""
	cases["'single'"] = "\"'single'\""
	cases["\"double\""] = "\"\\\"double\\\"\""

	checkConversion(c, cases, t)
}

func TestConvertScale(t *testing.T) {
	c := Conversion{Kind: "float", Scale: 2.0, Precision: 1}
	cases := make(map[string]string, 3)
	cases["1"] = "2.0"
	cases["-1"] = "-2.0"
	cases["0"] = "0.0"

	checkConversion(c, cases, t)

	c = Conversion{Kind: "float", Scale: 0.1, Precision: 1}
	cases = make(map[string]string, 5)
	cases["1"] = "0.1"
	cases["-1"] = "-0.1"
	cases["0"] = "0.0"
	cases["1.1"] = "0.1"
	cases["1.6"] = "0.2"

	checkConversion(c, cases, t)
}

func TestConvertBoolean(t *testing.T) {
	c := Conversion{Kind: "boolean"}
	cases := make(map[string]string, 8)
	cases["1"] = "true"
	cases["0"] = "false"
	cases["true"] = "true"
	cases["false"] = "false"
	cases["TRUE"] = "true"
	cases["FALSE"] = "false"
	cases["TruE"] = "true"
	cases["fAlsE"] = "false"
	cases[" true"] = "true"
	cases["false "] = "false"

	checkConversion(c, cases, t)

	expectedErrors := make([]string, 7)
	expectedErrors[0] = ""
	expectedErrors[1] = "foo"
	expectedErrors[2] = "01"
	expectedErrors[3] = "00"
	expectedErrors[4] = "yes"
	expectedErrors[5] = "no"
	expectedErrors[6] = "t r u e"

	checkExpectedErrors(c, expectedErrors, t)
}

func TestConvertOnOff(t *testing.T) {
	c := Conversion{Kind: "on-off"}
	cases := make(map[string]string, 8)
	cases["on"] = "true"
	cases["off"] = "false"
	cases["ON"] = "true"
	cases["OFF"] = "false"
	cases["oN"] = "true"
	cases["oFf"] = "false"
	cases[" on"] = "true"
	cases["off "] = "false"

	checkConversion(c, cases, t)

	expectedErrors := make([]string, 5)
	expectedErrors[0] = ""
	expectedErrors[1] = "foo"
	expectedErrors[2] = "of"
	expectedErrors[3] = "yes"
	expectedErrors[4] = "no"

	checkExpectedErrors(c, expectedErrors, t)
}

func checkConversion(c Conversion, cases map[string]string, t *testing.T) {
	for raw, expected := range cases {
		result, err := c.Convert(raw)
		if err != nil {
			t.Errorf("Converting %v: %v", raw, err)
		} else if result != expected {
			t.Errorf("Converting %v: %v != %v", raw, result, expected)
		}
	}
}

func checkExpectedErrors(c Conversion, cases []string, t *testing.T) {
	for _, raw := range cases {
		result, err := c.Convert(raw)
		if err == nil {
			t.Errorf("Converting %v: expected error, got '%v'", raw, result)
		}
	}
}
