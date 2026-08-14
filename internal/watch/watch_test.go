package watch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestBuildContextAndPass(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	issues := []Issue{{Number: 7, Title: "Bug", State: "OPEN"}}
	ctxText, err := BuildContext(root, issues)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctxText, "#7") || !strings.Contains(ctxText, "Lead") {
		t.Fatal(ctxText)
	}

	fake := &opencodeclient.FakeRunner{}
	ok, summary, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     true,
		Lister:      StaticLister{Issues: issues},
		Runner:      fake,
	})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(fake.Calls) != 1 {
		t.Fatal(fake.Calls)
	}
	if !strings.Contains(summary, "issues=1") {
		t.Fatal(summary)
	}

	_, triage, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     false,
		Lister:      StaticLister{Issues: issues},
	})
	if err != nil || !strings.Contains(triage, "#7") {
		t.Fatalf("%v %s", err, triage)
	}
}

func TestOvernightSkipsExecute(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	night := time.Date(2026, 8, 14, 20, 0, 0, 0, time.Local)
	fake := &opencodeclient.FakeRunner{}
	ok, summary, err := Pass(context.Background(), Options{
		ProjectRoot:    root,
		Execute:        true,
		Lister:         StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
		Runner:         fake,
		OvernightStart: "18:00",
		OvernightEnd:   "08:00",
		Now:            func() time.Time { return night },
	})
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v %s", ok, err, summary)
	}
	if len(fake.Calls) != 0 {
		t.Fatal("should not execute overnight")
	}
	if !strings.Contains(summary, "overnight") {
		t.Fatal(summary)
	}

	day := time.Date(2026, 8, 14, 10, 0, 0, 0, time.Local)
	ok, _, err = Pass(context.Background(), Options{
		ProjectRoot:    root,
		Execute:        true,
		Lister:         StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
		Runner:         fake,
		OvernightStart: "18:00",
		OvernightEnd:   "08:00",
		Now:            func() time.Time { return day },
	})
	if err != nil || !ok {
		t.Fatalf("day ok=%v err=%v", ok, err)
	}
	if len(fake.Calls) != 1 {
		t.Fatal(fake.Calls)
	}
}

func TestInOvernightWindow(t *testing.T) {
	night := time.Date(2026, 8, 14, 20, 30, 0, 0, time.Local)
	ok, err := InOvernight(night, "18:00", "08:00")
	if err != nil || !ok {
		t.Fatalf("night %v %v", ok, err)
	}
	morning := time.Date(2026, 8, 14, 3, 0, 0, 0, time.Local)
	ok, err = InOvernight(morning, "01:00", "05:00")
	if err != nil || !ok {
		t.Fatalf("same-day %v %v", ok, err)
	}
	noon := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	ok, err = InOvernight(noon, "18:00", "08:00")
	if err != nil || ok {
		t.Fatalf("midday should work %v %v", ok, err)
	}
	ok, err = InOvernight(noon, "", "")
	if err != nil || ok {
		t.Fatalf("unset %v %v", ok, err)
	}
}

func TestLoopOnce(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	err := Loop(context.Background(), Options{
		ProjectRoot: root,
		Once:        true,
		Lister:      StaticLister{Issues: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
}
