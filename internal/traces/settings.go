package traces

import (
	"fmt"
	"strings"

	"github.com/xeaser/squad-opencode/internal/squad"
)

const (
	ProtocolHTTP = "http/protobuf"
	ProtocolGRPC = "grpc"
)

type Settings struct {
	Endpoint string
	Protocol string
	Capture  bool
}

func ParseCapture(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func ResolveSettings(cfg *squad.Config, getenv func(string) string) (Settings, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	s := Settings{Protocol: ProtocolHTTP}

	if v := firstNonEmpty(getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"), getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); v != "" {
		s.Endpoint = v
	} else if cfg != nil && cfg.OTLP != nil {
		s.Endpoint = strings.TrimSpace(cfg.OTLP.Endpoint)
	}

	if v := firstNonEmpty(getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"), getenv("OTEL_EXPORTER_OTLP_PROTOCOL")); v != "" {
		s.Protocol = v
	} else if cfg != nil && cfg.OTLP != nil && strings.TrimSpace(cfg.OTLP.Protocol) != "" {
		s.Protocol = strings.TrimSpace(cfg.OTLP.Protocol)
	}

	if v := getenv("OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT"); v != "" {
		s.Capture = ParseCapture(v)
	} else if cfg != nil && cfg.OTLP != nil && cfg.OTLP.CaptureContent != nil {
		s.Capture = *cfg.OTLP.CaptureContent
	}

	switch s.Protocol {
	case ProtocolHTTP, ProtocolGRPC:
	default:
		return Settings{}, fmt.Errorf("unknown OTLP protocol %q (want %s or %s)", s.Protocol, ProtocolHTTP, ProtocolGRPC)
	}
	return s, nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
