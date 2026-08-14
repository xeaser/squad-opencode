package watch

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/traces"
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
	labeled, err := BuildContext(root, issues, "bug")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(labeled, "bug") {
		t.Fatal(labeled)
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

	spans, err := traces.List(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 {
		t.Fatalf("execute should record one span, got %+v", spans)
	}
	if spans[0].Name != "squad-oc.watch.execute" || spans[0].Status != "OK" {
		t.Fatalf("%+v", spans[0])
	}
	if spans[0].Attributes["issues"] != "1" {
		t.Fatalf("issues attr: %+v", spans[0].Attributes)
	}
}

func TestPassNoExecuteDoesNotRecordSpan(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	_, _, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     false,
		Lister:      StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	spans, err := traces.List(root, 10)
	if err != nil || len(spans) != 0 {
		t.Fatalf("triage should not record: %+v %v", spans, err)
	}
}

func TestPassExecuteRecordsErrorSpan(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	_, _, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     true,
		Lister: StaticLister{Issues: []Issue{
			{Number: 1, Title: "x", State: "OPEN"},
			{Number: 2, Title: "y", State: "OPEN"},
		}},
		Runner: &opencodeclient.FakeRunner{Err: errors.New("boom")},
	})
	if err == nil {
		t.Fatal("want execute error")
	}
	spans, err := traces.List(root, 10)
	if err != nil || len(spans) != 1 {
		t.Fatalf("%+v %v", spans, err)
	}
	if spans[0].Name != "squad-oc.watch.execute" || spans[0].Status != "ERROR" {
		t.Fatalf("%+v", spans[0])
	}
	if spans[0].Attributes["issues"] != "2" {
		t.Fatalf("issues attr: %+v", spans[0].Attributes)
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

func TestFormatHealthZero(t *testing.T) {
	now := time.Date(2026, 8, 14, 14, 32, 0, 0, time.Local)
	got := FormatHealth(Health{}, now)
	want := "" +
		"Ralph watch\n" +
		"\n" +
		"PID: n/a\n" +
		"Last poll: never\n" +
		"Last: \n" +
		"Next poll: not scheduled\n" +
		"Round: 0\n"
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
	if h.StartedAt.IsZero() {
		t.Fatal("startedAt should be set on first writePassHealth")
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

func TestPassesNotifyVerboseAtEveryLevel(t *testing.T) {
	cases := []struct {
		notify  NotifyLevel
		verbose bool
		level   NotifyLevel
		want    bool
	}{
		{NotifyNone, false, NotifyAll, false},
		{NotifyNone, false, NotifyImportant, false},
		{NotifyNone, true, NotifyAll, true},
		{NotifyNone, true, NotifyImportant, false},
		{NotifyImportant, false, NotifyAll, false},
		{NotifyImportant, false, NotifyImportant, true},
		{NotifyImportant, true, NotifyAll, true},
		{NotifyImportant, true, NotifyImportant, true},
		{NotifyAll, false, NotifyAll, true},
		{NotifyAll, false, NotifyImportant, true},
		{NotifyAll, true, NotifyAll, true},
	}
	for _, tc := range cases {
		got := passesNotify(Options{Notify: tc.notify, Verbose: tc.verbose}, tc.level)
		if got != tc.want {
			t.Errorf("notify=%v verbose=%v level=%v: got %v want %v",
				tc.notify, tc.verbose, tc.level, got, tc.want)
		}
	}
}

func TestNotifyNoneSkipsTriageDump(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	var logs []string
	_, _, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     false,
		Lister:      StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
		Notify:      NotifyNone,
		Logger: func(_ NotifyLevel, msg string) {
			logs = append(logs, msg)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range logs {
		if strings.Contains(msg, "#1") || strings.Contains(msg, "Issues") {
			t.Fatalf("triage dump logged under NotifyNone: %q", msg)
		}
	}
}

func TestNotifyImportantLogsExecuteError(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	var logs []string
	_, _, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     true,
		Lister:      StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
		Runner:      &opencodeclient.FakeRunner{Err: errors.New("boom")},
		Notify:      NotifyImportant,
		Logger: func(_ NotifyLevel, msg string) {
			logs = append(logs, msg)
		},
	})
	if err == nil {
		t.Fatal("want execute error")
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "boom") {
		t.Fatalf("expected execute error in logs: %v", logs)
	}
}

func TestLogFileAppends(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "nested", "watch.log")
	_, _, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     true,
		Lister:      StaticLister{Issues: []Issue{{Number: 1, Title: "x", State: "OPEN"}}},
		Runner:      &opencodeclient.FakeRunner{Err: errors.New("boom")},
		Notify:      NotifyImportant,
		LogFile:     logPath,
	})
	if err == nil {
		t.Fatal("want execute error")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "boom") {
		t.Fatalf("log file: %s", data)
	}
}

type recordingEscalator struct {
	calls []string
}

func (r *recordingEscalator) Reset(context.Context) error {
	r.calls = append(r.calls, "reset")
	return nil
}

func (r *recordingEscalator) ReprobeAuth(context.Context) error {
	r.calls = append(r.calls, "auth")
	return nil
}

func (r *recordingEscalator) GitPull(context.Context) error {
	r.calls = append(r.calls, "pull")
	return nil
}

type failLister struct {
	n      int
	cancel context.CancelFunc
}

func (f *failLister) List(context.Context) ([]Issue, error) {
	f.n++
	if f.n >= 4 && f.cancel != nil {
		f.cancel()
	}
	return nil, errors.New("list fail")
}

func TestNextTier(t *testing.T) {
	if NextTier(1) != 1 || NextTier(2) != 2 || NextTier(3) != 3 || NextTier(4) != 4 {
		t.Fatal("1..4")
	}
	if NextTier(0) != 1 || NextTier(9) != 4 {
		t.Fatal("clamp")
	}
}

func TestLoopEscalationTiers(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	esc := &recordingEscalator{}
	lister := &failLister{cancel: cancel}
	var backs []time.Duration
	err := Loop(ctx, Options{
		ProjectRoot: root,
		Once:        false,
		Interval:    time.Millisecond,
		Backoff:     15 * time.Millisecond,
		Lister:      lister,
		Escalator:   esc,
		Sleep: func(d time.Duration) {
			backs = append(backs, d)
		},
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	want := []string{"reset", "auth", "pull"}
	if strings.Join(esc.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls %v want %v", esc.calls, want)
	}
	found := false
	for _, d := range backs {
		if d == 15*time.Millisecond {
			found = true
		}
	}
	if !found {
		t.Fatalf("want long backoff, got %v", backs)
	}
}

func TestLoopOnceDoesNotEscalate(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	esc := &recordingEscalator{}
	err := Loop(context.Background(), Options{
		ProjectRoot: root,
		Once:        true,
		Lister:      StaticLister{Err: errors.New("list fail")},
		Escalator:   esc,
	})
	if err == nil {
		t.Fatal("want pass error")
	}
	if len(esc.calls) != 0 {
		t.Fatalf("escalated on --once: %v", esc.calls)
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	run("commit", "--allow-empty", "-m", "init")
	return dir
}

func TestGitNotesBackendRoundTrip(t *testing.T) {
	dir := initTempRepo(t)
	b := GitNotesBackend{Dir: dir}
	want := Health{PID: 7, Round: 3, LastSummary: "issues=1 execute=false"}
	if err := b.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := b.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != 7 || got.Round != 3 || got.LastSummary != want.LastSummary {
		t.Fatalf("%+v", got)
	}
}

func TestGitNotesBackendFreshHEAD(t *testing.T) {
	dir := initTempRepo(t)
	got, err := (GitNotesBackend{Dir: dir}).Load(context.Background())
	if err != nil || got.PID != 0 || got.Round != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestGitNotesBackendRequiresRepo(t *testing.T) {
	err := (GitNotesBackend{Dir: t.TempDir()}).Save(context.Background(), Health{PID: 1})
	if err == nil {
		t.Fatal("want repo error")
	}
}

func TestOrphanBranchBackendRoundTrip(t *testing.T) {
	dir := initTempRepo(t)
	headCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	headCmd.Dir = dir
	before, err := headCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	b := OrphanBranchBackend{Dir: dir}
	want := Health{PID: 11, Round: 4, LastSummary: "ok"}
	if err := b.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := b.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != 11 || got.Round != 4 || got.LastSummary != want.LastSummary {
		t.Fatalf("%+v", got)
	}
	headCmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	headCmd.Dir = dir
	after, err := headCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(after)) != strings.TrimSpace(string(before)) {
		t.Fatalf("worktree moved: %s -> %s", before, after)
	}
	if strings.Contains(string(after), "ralph-state") {
		t.Fatal("checked out orphan branch")
	}
}

func TestMemoryBackend(t *testing.T) {
	m := &MemoryBackend{}
	if err := m.Save(context.Background(), Health{PID: 1, Round: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := m.Load(context.Background())
	if err != nil || got.PID != 1 || got.Round != 2 {
		t.Fatalf("%+v %v", got, err)
	}
}
