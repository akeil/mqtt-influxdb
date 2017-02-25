package mqttinflux

import (
	"strings"
	"testing"
)

func TestMeasurementName(t *testing.T) {
	m := NewMeasurement("m & m")
	err := m.Validate()
	if err == nil {
		t.Error("Expected error for invalid measurement name")
	}
}

func TestMeasurementTag(t *testing.T) {
	m := NewMeasurement("m")
	m.SetValue("1")
	m.Tag("foo", "bar")
	s := m.Format()

	if !strings.Contains(s, "foo=bar") {
		t.Fail()
	}

	// invalid tag value
	m.Tag("foo=bar", "baz")
	err := m.Validate()
	if err == nil {
		t.Error("Expected error for invalid tag name")
	}

	// invalid tag name
	m = NewMeasurement("m")
	m.SetValue("1")
	m.Tag("foo", "baz=bar")
	err = m.Validate()
	if err == nil {
		t.Error("Expected error for invalid tag value")
	}
}

func TestMeasurementValue(t *testing.T) {
	m := NewMeasurement("m")

	err := m.Validate()
	if err == nil {
		t.Error("Expected error for missing value")
	}

	m.SetValue("222 1234")
	err = m.Validate()
	if err == nil {
		t.Error("Expected error for invalid value")
	}

	m.SetValue("")
	err = m.Validate()
	if err != nil {
		t.Errorf("Expected OK, got %v", err)
	}

	m.SetValue("222")
	err = m.Validate()
	if err != nil {
		t.Errorf("Expected OK, got %v", err)
	}
}
