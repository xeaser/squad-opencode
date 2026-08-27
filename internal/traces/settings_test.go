package traces

import (
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestResolveSettingsEnvWinsAndDefaults(t *testing.T) {
	capTrue := true
	cfg := &squad.Config{OTLP: &squad.OTLPConfig{
		Endpoint:       "http://from-file:4318",
		Protocol:       "grpc",
		CaptureContent: &capTrue,
	}}
	env := map[string]string{
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT":                 "http://from-traces-env:4318/v1/traces",
		"OTEL_EXPORTER_OTLP_ENDPOINT":                        "http://from-generic-env:4318",
		"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL":                 "http/protobuf",
		"OTEL_EXPORTER_OTLP_PROTOCOL":                        "grpc",
		"OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT": "false",
	}
	getenv := func(k string) string { return env[k] }
	s, err := ResolveSettings(cfg, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if s.Endpoint != "http://from-traces-env:4318/v1/traces" {
		t.Fatalf("endpoint %q", s.Endpoint)
	}
	if s.Protocol != "http/protobuf" {
		t.Fatalf("protocol %q", s.Protocol)
	}
	if s.Capture {
		t.Fatal("capture env false must win")
	}

	s, err = ResolveSettings(nil, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if s.Endpoint != "" || s.Protocol != "http/protobuf" || s.Capture {
		t.Fatalf("defaults %+v", s)
	}
}

func TestResolveSettingsConfigWhenEnvEmpty(t *testing.T) {
	cfg := &squad.Config{OTLP: &squad.OTLPConfig{
		Endpoint: "http://127.0.0.1:3000/api/public/otel",
		Protocol: "http/protobuf",
	}}
	s, err := ResolveSettings(cfg, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if s.Endpoint != "http://127.0.0.1:3000/api/public/otel" || s.Capture {
		t.Fatalf("%+v", s)
	}
}

func TestResolveSettingsBadProtocol(t *testing.T) {
	_, err := ResolveSettings(&squad.Config{OTLP: &squad.OTLPConfig{Protocol: "http/json"}}, func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), "protocol") {
		t.Fatalf("got %v", err)
	}
}

func TestParseCapture(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "1", "yes", "Yes"} {
		if !ParseCapture(v) {
			t.Fatalf("%q should be on", v)
		}
	}
	for _, v := range []string{"", "false", "0", "no", "off"} {
		if ParseCapture(v) {
			t.Fatalf("%q should be off", v)
		}
	}
}
