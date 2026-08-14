package opencodeclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/traces"
)

func TestFakeRunner(t *testing.T) {
	f := &FakeRunner{Text: "hello"}
	res, err := f.Run(context.Background(), RunRequest{Prompt: "p", Agent: "squad"})
	if err != nil || res.Text != "hello" || len(f.Calls) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestRecordRunSpan(t *testing.T) {
	root := t.TempDir()
	start := time.Now().Add(-20 * time.Millisecond)
	recordRun(RunRequest{Directory: root, Agent: "lead", Prompt: "abcd"}, start, nil)
	spans, err := traces.List(root, 10)
	if err != nil || len(spans) != 1 {
		t.Fatalf("%+v %v", spans, err)
	}
	s := spans[0]
	if s.Name != "squad-oc.run" || s.Status != "OK" {
		t.Fatalf("%+v", s)
	}
	if s.Attributes["agent"] != "lead" || s.Attributes["prompt_bytes"] != "4" {
		t.Fatalf("attrs %+v", s.Attributes)
	}

	recordRun(RunRequest{Directory: root, Prompt: "xy"}, start, errors.New("boom"))
	spans, err = traces.List(root, 10)
	if err != nil || len(spans) != 2 {
		t.Fatalf("%+v %v", spans, err)
	}
	if spans[1].Status != "ERROR" || spans[1].Attributes["agent"] != "squad" {
		t.Fatalf("%+v", spans[1])
	}
}
