package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/xeaser/squad-opencode/internal/version"
)

// NewExporterKind returns the protocol kind without starting an exporter.
func NewExporterKind(protocol string) (string, error) {
	switch protocol {
	case ProtocolGRPC:
		return ProtocolGRPC, nil
	case ProtocolHTTP, "":
		return ProtocolHTTP, nil
	default:
		return "", fmt.Errorf("unknown OTLP protocol %q (want %s or %s)", protocol, ProtocolHTTP, ProtocolGRPC)
	}
}

// Push exports parent/child via the official OTel SDK. Empty endpoint is a no-op.
func Push(ctx context.Context, s Settings, parent Span, child *Span) error {
	if s.Endpoint == "" {
		return nil
	}
	exp, err := newExporter(ctx, s)
	if err != nil {
		return err
	}
	res := resource.NewSchemaless(
		attribute.String("service.name", "squad-oc"),
		attribute.String("service.version", version.Version),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithResource(res),
	)
	recErr := recordWithTracer(ctx, tp.Tracer("squad-oc"), parent, child, s.Capture)
	shutErr := tp.Shutdown(ctx)
	if recErr != nil {
		return recErr
	}
	return shutErr
}

func newExporter(ctx context.Context, s Settings) (sdktrace.SpanExporter, error) {
	kind, err := NewExporterKind(s.Protocol)
	if err != nil {
		return nil, err
	}
	if kind == ProtocolGRPC {
		return newGRPCExporter(ctx, s.Endpoint)
	}
	return newHTTPExporter(ctx, s.Endpoint)
}

func httpTracesURL(endpoint string) string {
	if strings.HasSuffix(endpoint, "/v1/traces") {
		return endpoint
	}
	return strings.TrimRight(endpoint, "/") + "/v1/traces"
}

func newHTTPExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(httpTracesURL(endpoint)))
}

func newGRPCExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	hostport, insecure := grpcTarget(endpoint)
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(hostport)}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	return otlptracegrpc.New(ctx, opts...)
}

func grpcTarget(endpoint string) (hostport string, insecure bool) {
	u := strings.TrimSpace(endpoint)
	lower := strings.ToLower(u)
	switch {
	case strings.HasPrefix(lower, "http://"):
		insecure = true
		u = u[len("http://"):]
	case strings.HasPrefix(lower, "https://"):
		u = u[len("https://"):]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	host := u
	if h, _, err := net.SplitHostPort(u); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1" {
		insecure = true
	}
	return u, insecure
}

func recordWithTracer(ctx context.Context, tr trace.Tracer, parent Span, child *Span, capture bool) error {
	pctx, pspan := tr.Start(ctx, parent.Name, startOpts(parent.Start, parentOTelAttrs(parent))...)
	if strings.EqualFold(parent.Status, "ERROR") {
		pspan.SetStatus(codes.Error, "")
	}
	if child != nil {
		_, cspan := tr.Start(pctx, child.Name, startOpts(child.Start, childOTelAttrs(*child, capture))...)
		if strings.EqualFold(child.Status, "ERROR") {
			cspan.SetStatus(codes.Error, "")
		}
		endSpan(cspan, child.End)
	}
	endSpan(pspan, parent.End)
	return nil
}

func startOpts(start time.Time, attrs []attribute.KeyValue) []trace.SpanStartOption {
	opts := make([]trace.SpanStartOption, 0, 2)
	if !start.IsZero() {
		opts = append(opts, trace.WithTimestamp(start))
	}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	return opts
}

func endSpan(sp trace.Span, end time.Time) {
	if !end.IsZero() {
		sp.End(trace.WithTimestamp(end))
	} else {
		sp.End()
	}
}

func parentOTelAttrs(s Span) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if s.Agent != "" {
		attrs = append(attrs, attribute.String("gen_ai.agent.name", s.Agent))
	}
	if s.SessionID != "" {
		attrs = append(attrs, attribute.String("gen_ai.conversation.id", s.SessionID))
		attrs = append(attrs, attribute.String("session.id", s.SessionID))
	}
	if v := s.Attributes["issues"]; v != "" {
		attrs = append(attrs, attribute.String("issues", v))
	}
	return attrs
}

func childOTelAttrs(s Span, capture bool) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gen_ai.operation.name", "chat"),
		attribute.Int64("gen_ai.usage.input_tokens", int64(s.InputTokens)),
		attribute.Int64("gen_ai.usage.output_tokens", int64(s.OutputTokens)),
		attribute.Float64("gen_ai.usage.cost", s.Cost),
	}
	if s.Provider != "" {
		attrs = append(attrs, attribute.String("gen_ai.provider.name", s.Provider))
	}
	if s.Model != "" {
		attrs = append(attrs, attribute.String("gen_ai.request.model", s.Model))
	}
	if s.ReasoningTokens != 0 {
		attrs = append(attrs, attribute.Int64("gen_ai.usage.reasoning.output_tokens", int64(s.ReasoningTokens)))
	}
	if s.CacheReadTokens != 0 {
		attrs = append(attrs, attribute.Int64("gen_ai.usage.cache_read.input_tokens", int64(s.CacheReadTokens)))
	}
	if s.CacheWriteTokens != 0 {
		attrs = append(attrs, attribute.Int64("gen_ai.usage.cache_write.input_tokens", int64(s.CacheWriteTokens)))
	}
	if s.Agent != "" {
		attrs = append(attrs, attribute.String("gen_ai.agent.name", s.Agent))
	}
	if s.SessionID != "" {
		attrs = append(attrs, attribute.String("gen_ai.conversation.id", s.SessionID))
		attrs = append(attrs, attribute.String("session.id", s.SessionID))
	}
	if capture {
		attrs = append(attrs,
			attribute.String("gen_ai.input.messages", messagesJSON("user", s.Prompt)),
			attribute.String("gen_ai.output.messages", messagesJSON("assistant", s.Completion)),
		)
	}
	return attrs
}

func messagesJSON(role, content string) string {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	b, err := json.Marshal([]msg{{Role: role, Content: content}})
	if err != nil {
		return ""
	}
	return string(b)
}
