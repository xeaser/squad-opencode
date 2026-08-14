package watch

import (
	"context"
	"os"
	"path/filepath"
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
	nightHealth, herr := ReadHealth(root)
	if herr != nil || !nightHealth.Overnight || !strings.Contains(nightHealth.LastSummary, "overnight") {
		t.Fatalf("overnight health %+v %v", nightHealth, herr)
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

func TestReadWriteHealth(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 14, 14, 32, 0, 0, time.Local)
	h := Health{
		PID:         99,
		StartedAt:   now.Add(-time.Hour),
		LastPoll:    now,
		LastSummary: "issues=1 execute=false",
		Round:       3,
		NextPoll:    now.Add(10 * time.Minute),
	}
	if err := WriteHealth(root, h); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(root, ".squad", "ralph-status.json")
	if StatusPath(root) != wantPath {
		t.Fatal(StatusPath(root))
	}
	got, err := ReadHealth(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != 99 || got.Round != 3 || got.LastSummary != h.LastSummary {
		t.Fatalf("%+v", got)
	}
	if !got.LastPoll.Equal(h.LastPoll) {
		t.Fatalf("poll %v %v", got.LastPoll, h.LastPoll)
	}
}

func TestReadHealthMissing(t *testing.T) {
	_, err := ReadHealth(t.TempDir())
	if !os.IsNotExist(err) {
		t.Fatalf("want missing, got %v", err)
	}
}

func TestFormatHealth(t *testing.T) {
	now := time.Date(2026, 8, 14, 14, 32, 0, 0, time.Local)
	h := Health{
		PID:         12345,
		StartedAt:   now.Add(-2*time.Hour - 15*time.Minute),
		LastPoll:    now.Add(-2 * time.Minute),
		LastSummary: "issues=3 execute=true",
		NextPoll:    now.Add(3 * time.Minute),
		Round:       42,
	}
	got := FormatHealth(h, now)
	want := "" +
		"Ralph watch\n" +
		"\n" +
		"PID: 12345\n" +
		"Uptime: 2h 15m\n" +
		"Last poll: 2 minutes ago\n" +
		"Last: issues=3 execute=true\n" +
		"Next poll: 14:35 (in 3 minutes)\n" +
		"Round: 42\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPassWritesHealth(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	_, summary, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     false,
		Lister:      StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := ReadHealth(root)
	if err != nil {
		t.Fatal(err)
	}
	if h.Round != 1 {
		t.Fatalf("round %d", h.Round)
	}
	if !strings.Contains(h.LastSummary, "issues=1") {
		t.Fatalf("%s / %s", h.LastSummary, summary)
	}
	if h.Overnight {
		t.Fatal("not overnight")
	}
}

func TestLoopSetsPIDOnFirstWrite(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := Loop(context.Background(), Options{
		ProjectRoot: root,
		Once:        true,
		Lister:      StaticLister{Issues: nil},
	}); err != nil {
		t.Fatal(err)
	}
	h, err := ReadHealth(root)
	if err != nil {
		t.Fatal(err)
	}
	if h.PID != os.Getpid() {
		t.Fatalf("pid %d want %d", h.PID, os.Getpid())
	}
	if h.StartedAt.IsZero() {
		t.Fatal("startedAt")
	}
	if h.Round < 1 {
		t.Fatalf("round %d", h.Round)
	}
	if h.NextPoll.IsZero() {
		t.Fatal("nextPoll")
	}
}
