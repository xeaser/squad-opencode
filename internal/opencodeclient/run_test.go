package opencodeclient

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/traces"
)

func TestMain(m *testing.M) {
	pushOTLP = func(context.Context, traces.Settings, traces.Span, *traces.Span) error {
		return nil
	}
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	os.Exit(m.Run())
}

func TestFakeRunner(t *testing.T) {
	f := &FakeRunner{Text: "hello"}
	res, err := f.Run(context.Background(), RunRequest{Prompt: "p", Agent: "squad"})
	if err != nil || res.Text != "hello" || len(f.Calls) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if !res.HasGeneration || res.SessionID != "fake-session" {
		t.Fatalf("generation %+v", res)
	}
}

func TestRecordRunSpan(t *testing.T) {
	root := t.TempDir()
	start := time.Now().Add(-20 * time.Millisecond)
	recordRun(RunRequest{Directory: root, Agent: "lead", Prompt: "abcd"}, start, nil, RunResult{
		SessionID: "ses_1", Text: "ok",
		HasGeneration: true, Provider: "xai", Model: "grok-4",
		InputTokens: 1, OutputTokens: 2, Cost: 0.1,
	})
	spans, err := traces.List(root, 10)
	if err != nil || len(spans) != 2 {
		t.Fatalf("%+v %v", spans, err)
	}
	if spans[0].Name != "squad-oc.run" || spans[1].Name != traces.NameChat {
		t.Fatalf("%+v", spans)
	}
	if spans[1].Model != "grok-4" || spans[1].Prompt != "abcd" || spans[1].Completion != "ok" {
		t.Fatalf("child %+v", spans[1])
	}

	recordRun(RunRequest{Directory: root, Prompt: "xy"}, start, errors.New("boom"), RunResult{})
	spans, err = traces.List(root, 10)
	if err != nil || len(spans) != 3 {
		t.Fatalf("%+v %v", spans, err)
	}
	if spans[2].Name != "squad-oc.run" || spans[2].Status != "ERROR" {
		t.Fatalf("%+v", spans[2])
	}
}

func TestRecordRunCollectorFailureDoesNotFail(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	prev := pushOTLP
	t.Cleanup(func() { pushOTLP = prev })
	pushOTLP = func(context.Context, traces.Settings, traces.Span, *traces.Span) error {
		return errors.New("collector down")
	}
	start := time.Now()
	recordRun(RunRequest{Directory: root, Prompt: "p"}, start, nil, RunResult{
		SessionID: "s", Text: "ok", HasGeneration: true,
	})
	spans, err := traces.List(root, 10)
	if err != nil || len(spans) != 2 {
		t.Fatalf("JSONL must still be written: %+v %v", spans, err)
	}
	if spans[0].Name != "squad-oc.run" || spans[1].Name != traces.NameChat {
		t.Fatalf("%+v", spans)
	}
}
