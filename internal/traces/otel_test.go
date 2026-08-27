package traces

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBuildParentChildAndParentOnly(t *testing.T) {
	start := time.Date(2026, 8, 27, 15, 0, 0, 0, time.UTC)
	parent, child := Build(RecordInput{
		ParentName:    "squad-oc.run",
		Start:         start,
		End:           start.Add(time.Second),
		Agent:         "lead",
		Prompt:        "hi",
		Completion:    "yo",
		SessionID:     "ses_1",
		Attrs:         map[string]string{"agent": "lead", "prompt_bytes": "2"},
		HasGeneration: true,
		Provider:      "xai",
		Model:         "grok-4",
		InputTokens:   1,
		OutputTokens:  2,
		Cost:          0,
	})
	if parent.Name != "squad-oc.run" || child == nil || child.Name != NameChat {
		t.Fatalf("%+v %+v", parent, child)
	}
	if child.TraceID != parent.TraceID || child.ParentID != parent.SpanID {
		t.Fatal("tree")
	}
	if child.Prompt != "hi" || child.Cost != 0 || child.Model != "grok-4" {
		t.Fatalf("child %+v", child)
	}

	parent, child = Build(RecordInput{ParentName: "squad-oc.run", Err: errors.New("boom"), Agent: "squad"})
	if parent.Status != "ERROR" || child != nil {
		t.Fatalf("fail %+v %v", parent, child)
	}
}

func TestPushEmptyEndpointNoop(t *testing.T) {
	if err := Push(context.Background(), Settings{}, Span{Name: "squad-oc.run"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestWriteAppendsAndReturnsPushError(t *testing.T) {
	root := t.TempDir()
	start := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)
	pushErr := errors.New("collector down")
	err := Write(root, RecordInput{
		ParentName:    "squad-oc.run",
		Start:         start,
		End:           start.Add(time.Millisecond),
		Agent:         "lead",
		Prompt:        "hi",
		Completion:    "yo",
		SessionID:     "ses_1",
		HasGeneration: true,
		Model:         "grok-4",
	}, Settings{Endpoint: "http://127.0.0.1:1"}, func(context.Context, Settings, Span, *Span) error {
		return pushErr
	})
	if !errors.Is(err, pushErr) {
		t.Fatalf("want push error, got %v", err)
	}
	if !strings.Contains(err.Error(), "otlp push:") {
		t.Fatalf("want otlp push wrap, got %v", err)
	}
	spans, err := List(root, 10)
	if err != nil || len(spans) != 2 {
		t.Fatalf("JSONL %+v %v", spans, err)
	}
	if spans[0].Name != "squad-oc.run" || spans[1].Name != NameChat {
		t.Fatalf("%+v", spans)
	}

	empty := t.TempDir()
	if err := Write(empty, RecordInput{ParentName: "squad-oc.run"}, Settings{}, nil); err != nil {
		t.Fatal(err)
	}
	spans, err = List(empty, 10)
	if err != nil || len(spans) != 1 {
		t.Fatalf("no-endpoint still JSONL: %+v %v", spans, err)
	}
}

func TestWriteWrapsAppendError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".squad"), []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Write(root, RecordInput{ParentName: "squad-oc.run"}, Settings{}, nil)
	if err == nil || !strings.Contains(err.Error(), "append:") {
		t.Fatalf("want append wrap, got %v", err)
	}
}

func TestRecordToSpanRecorderTypedAttrs(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	parent, child := Build(RecordInput{
		ParentName: "squad-oc.run", Agent: "squad", SessionID: "ses_1",
		HasGeneration: true, Provider: "xai", Model: "grok-4",
		InputTokens: 3, OutputTokens: 4, Cost: 0.5,
		Prompt: "P", Completion: "C",
	})
	if err := recordWithTracer(context.Background(), tp.Tracer("squad-oc"), parent, child, false); err != nil {
		t.Fatal(err)
	}
	spans := rec.Ended()
	if len(spans) != 2 {
		t.Fatalf("len=%d", len(spans))
	}
	// order: child ends first if started as child; accept either order, find by name
	var gen sdktrace.ReadOnlySpan
	for _, sp := range spans {
		if sp.Name() == NameChat {
			gen = sp
		}
	}
	if gen == nil {
		t.Fatal("missing gen_ai.chat")
	}
	attrs := attrMap(gen.Attributes())
	if attrs["gen_ai.request.model"] != "grok-4" {
		t.Fatalf("%v", attrs)
	}
	if _, ok := attrs["gen_ai.input.messages"]; ok {
		t.Fatal("capture off")
	}
	// tokens must be int64-typed on the span (check via attribute.KeyValue)
	if !hasInt(gen.Attributes(), "gen_ai.usage.input_tokens", 3) {
		t.Fatalf("tokens %+v", gen.Attributes())
	}
}

func TestRecordToSpanRecorderCaptureOn(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	parent, child := Build(RecordInput{
		ParentName: "squad-oc.run", Agent: "squad", SessionID: "ses_1",
		HasGeneration: true, Provider: "xai", Model: "grok-4",
		InputTokens: 3, OutputTokens: 4, Cost: 0.5,
		Prompt: "P", Completion: "C",
	})
	if err := recordWithTracer(context.Background(), tp.Tracer("squad-oc"), parent, child, true); err != nil {
		t.Fatal(err)
	}
	var gen sdktrace.ReadOnlySpan
	for _, sp := range rec.Ended() {
		if sp.Name() == NameChat {
			gen = sp
		}
	}
	if gen == nil {
		t.Fatal("missing gen_ai.chat")
	}
	attrs := attrMap(gen.Attributes())
	inMsg, ok := attrs["gen_ai.input.messages"]
	if !ok || !strings.Contains(inMsg, "P") {
		t.Fatalf("input messages %v", attrs)
	}
	outMsg, ok := attrs["gen_ai.output.messages"]
	if !ok || !strings.Contains(outMsg, "C") {
		t.Fatalf("output messages %v", attrs)
	}
}

func TestHTTPExporterHitsTestServer(t *testing.T) {
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = b
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	parent, child := Build(RecordInput{ParentName: "squad-oc.run", HasGeneration: true, Model: "m"})
	err := Push(context.Background(), Settings{Endpoint: srv.URL, Protocol: ProtocolHTTP}, parent, child)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no protobuf posted")
	}
}

func TestExporterKindGRPC(t *testing.T) {
	k, err := NewExporterKind(ProtocolGRPC)
	if err != nil || k != ProtocolGRPC {
		t.Fatalf("%s %v", k, err)
	}
}

func attrMap(kvs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}

func hasInt(kvs []attribute.KeyValue, key string, want int64) bool {
	for _, kv := range kvs {
		if string(kv.Key) == key && kv.Value.Type() == attribute.INT64 && kv.Value.AsInt64() == want {
			return true
		}
	}
	return false
}
