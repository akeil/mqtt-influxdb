package mqttinflux

import (
	"strings"
	"testing"
)

func TestMeasurementName(t *testing.T) {
	m := NewMeasurement("db", "m & m")
	err := m.Validate()
	if err == nil {
		t.Error("Expected error for invalid measurement name")
	}
}

func TestMeasurementDBName(t *testing.T) {
	m := NewMeasurement("?invalid db", "m")
	err := m.Validate()
	if err == nil {
		t.Error("Expected error for invalid database name")
	}
}

func TestMeasurementTag(t *testing.T) {
	m := NewMeasurement("db", "m")
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
	m = NewMeasurement("db", "m")
	m.SetValue("1")
	m.Tag("foo", "baz=bar")
	err = m.Validate()
	if err == nil {
		t.Error("Expected error for invalid tag value")
	}
}

func TestMeasurementValue(t *testing.T) {
	m := NewMeasurement("db", "m")

	err := m.Validate()
	if err == nil {
		t.Error("Expected error for missing value")
	}

	//m.SetValue("222 1234")
	//err = m.Validate()
	//if err == nil {
	//	t.Error("Expected error for invalid value")
	//}

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

// Template Context -----------------------------------------------------------

func TestTemplateTopic(t *testing.T) {
	tags := map[string]string{
		"invalid": "{{.Topic 4}}",
	}
	s := Subscription{
		Topic:       "foo/bar/baz",
		Measurement: "{{.Topic 2}}",
		Tags:        tags,
	}

	err := s.parseTemplates()
	if err != nil {
		t.Errorf("error parsing template: %v", err)
	}
	ctx := NewTemplateContext("foo/bar/baz", "123")

	result, err := s.fillTemplate("measurement", ctx)
	if err != nil {
		t.Errorf("fill template: %v", err)
	}
	if result != "baz" {
		t.Errorf("Expected %v, got %v", "baz", result)
	}

	result, err = s.fillTemplate("tag.invalid", ctx)
	if err == nil {
		t.Error("Expected error, got OK")
	}
}

func TestTemplateJSON(t *testing.T) {
	s := new(Subscription)
	s.Topic = "foo/bar/baz"
	s.Measurement = "something"
	s.Tags = map[string]string{
		"path":     "{{.JSON \"foo.bar\"}}",
		"nonexist": "{{.JSON \"foo.nonexist\"}}",
	}

	err := s.parseTemplates()
	if err != nil {
		t.Errorf("error parsing template: %v", err)
	}
	json := `{
	  "x": "y",
	  "foo": {
	    "bar": "value"
	  }
    }`
	ctx := NewTemplateContext("foo/bar/baz", json)

	result, err := s.fillTemplate("tag.path", ctx)
	if err != nil {
		t.Errorf("fill template: %v", err)
	}
	if result != "value" {
		t.Errorf("Expected %v, got %v", "value", result)
	}

	// non-existent json path
	result, err = s.fillTemplate("tag.nonexist", ctx)
	if err == nil {
		t.Error("Expected error, got OK")
	}

	// invalid JSON
	ctx2 := NewTemplateContext("foo/bar/baz", "this is not JSON")
	result, err = s.fillTemplate("tag.path", ctx2)
	if err == nil {
		t.Error("Expected error, got OK")
	}
}
