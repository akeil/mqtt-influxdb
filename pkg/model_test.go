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
	s := &Subscription{
		Topic:       "foo/bar/baz",
		Measurement: "{{.Topic 2}}",
		Tags:        tags,
	}

	err := s.parseTemplates()
	if err != nil {
		t.Errorf("error parsing template: %v", err)
	}
	ctx := NewTemplateContext(s, "foo/bar/baz", "123")

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
	jsonPayload := `{
	  "x": "y",
	  "foo": {
	    "bar": "value",
		"intvalue": 123,
		"arr": [1, 2, 3]
	  }
    }`
	ctx := NewTemplateContext(s, "foo/bar/baz", jsonPayload)

	cases := map[string]string{
		"foo.bar":      "value",
		"foo.intvalue": "123",
		"foo.arr.1":    "2",
	}

	for path, expected := range cases {
		result, parseErr := ctx.JSON(path)
		if parseErr != nil {
			t.Errorf("error reading JSON: %v", parseErr)
		}

		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	}

	_, err = ctx.JSON("doesnotexist.foo.bar")
	if err == nil {
		t.Error("Expected rror, got ok")
	}

	// invalid JSON
	ctx2 := NewTemplateContext(s, "foo/bar/baz", "this is not JSON")
	_, err = ctx2.JSON("tag.path")
	if err == nil {
		t.Error("Expected error, got OK")
	}
}

func TestTemplateCSV(t *testing.T) {
	s := new(Subscription)
	s.Topic = "foo/bar/baz"

	err := s.parseTemplates()
	if err != nil {
		t.Errorf("error parsing template: %v", err)
	}
	expect := []string{"123", "5.5", "abc"}
	csvPayload := "123,5.5,abc"
	ctx := NewTemplateContext(s, "foo/bar/baz", csvPayload)

	for index, expected := range expect {
		value, parseErr := ctx.CSV(index)
		if parseErr != nil {
			t.Errorf("error parsing csv: %v", parseErr)
		}
		if value != expected {
			t.Errorf("CSV: expected %q, got %q", expected, value)
		}
	}

	// index out of range
	_, err = ctx.CSV(4)
	if err == nil {
		t.Error("Expected error, got OK")
	}

}

func TestTemplateInvalidCSV(t *testing.T) {
	invalidCSV := NewTemplateContext(&Subscription{}, "foo/bar/baz", "")
	_, err := invalidCSV.CSV(0)
	if err == nil {
		t.Error("Expected error, got OK")
	}
}

func TestHandleCSV(t *testing.T) {
	s := &Subscription{
		Value:       "CSV 1",
		Measurement: "test",
		Conversion: Conversion{
			Kind:      "float",
			Precision: 1,
		},
	}

	m, err := s.Read("foo/bar", "123,456")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if m.Values["value"] != "456.0" {
		t.Errorf("expected 456, got %v", err)
	}
}

func TestCSVCustomSeparator(t *testing.T) {
	s := &Subscription{
		Value:        "CSV 1",
		Measurement:  "test",
		CSVSeparator: ";",
	}

	m, err := s.Read("foo/bar", "123;456")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if m.Values["value"] != "456" {
		t.Errorf("expected 456, got %v", err)
	}

	// invalid separator
	s.CSVSeparator = "++" // multiple chars
	_, err = s.Read("foo/bar", "123;456")
	if err == nil {
		t.Error("Expected error, got ok")
	}
}
