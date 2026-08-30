package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/xeaser/squad-opencode/internal/brief"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/traces"
)

func TestMain(m *testing.M) {
	tracesIngestPush = func(context.Context, traces.Settings, traces.Span, *traces.Span) error {
		return nil
	}
	_ = os.Unsetenv("OPENCODE_DB")
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	xdg, err := os.MkdirTemp("", "squad-cli-xdg-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.Setenv("XDG_DATA_HOME", xdg)
	code := m.Run()
	_ = os.RemoveAll(xdg)
	os.Exit(code)
}

func TestHelpAndUnknown(t *testing.T) {
	if Execute(nil) != 0 {
		t.Fatal("help")
	}
	if Execute([]string{"nope"}) != 2 {
		t.Fatal("unknown")
	}
	if Execute([]string{"version"}) != 0 {
		t.Fatal("version")
	}
}

func TestAliasDispatch(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if Execute([]string{"nope"}) != 2 {
		t.Fatal("unknown still 2")
	}
	// heartbeat/triage/loop must be recognized (not unknown exit 2)
	if code := Execute([]string{"heartbeat"}); code == 2 {
		t.Fatal("heartbeat should not be unknown")
	}
	if code := Execute([]string{"triage", "--once"}); code == 2 {
		t.Fatal("triage should not be unknown")
	}
	if code := Execute([]string{"loop", "--once"}); code == 2 {
		t.Fatal("loop should not be unknown")
	}
	if code := Execute([]string{"watch", "--label", "bug", "--once"}); code == 2 {
		t.Fatal("--label should not be unknown")
	}
	if code := Execute([]string{"watch", "--notify-level", "none", "--once"}); code == 2 {
		t.Fatal("--notify-level should not be unknown")
	}
	if Execute([]string{"watch", "--notify-level", "nope"}) != 2 {
		t.Fatal("invalid --notify-level should be 2")
	}
	if Execute([]string{"watch", "--state-backend", "nope"}) != 2 {
		t.Fatal("invalid --state-backend should be 2")
	}
	if code := Execute([]string{"watch", "--state-backend", "memory", "--health"}); code == 2 {
		t.Fatal("--state-backend memory should not be unknown")
	}
	if code := Execute([]string{"watch", "--force", "--once"}); code == 2 {
		t.Fatal("--force should not be unknown")
	}
	if code := Execute([]string{"watch", "--retry-label", "ralph-retry", "--once"}); code == 2 {
		t.Fatal("--retry-label should not be unknown")
	}
}

func TestInitExportImportViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "cli"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"status"}); code != 0 {
		t.Fatal("status")
	}
	if code := Execute([]string{"upgrade", "--dry-run"}); code != 0 {
		t.Fatal("upgrade")
	}
	snap := filepath.Join(root, "out.json")
	if code := Execute([]string{"export", snap}); code != 0 {
		t.Fatal("export")
	}
	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"import", snap}); code != 0 {
		t.Fatal("import")
	}
	b, err := os.ReadFile(filepath.Join(other, ".squad", "team.md"))
	if err != nil || !strings.Contains(string(b), "cli") {
		t.Fatalf("%v %s", err, b)
	}
	if _, err := os.Stat(filepath.Join(other, ".opencode", "agents", "lead.md")); !os.IsNotExist(err) {
		t.Fatal("default import should not write host files")
	}

	withHost := t.TempDir()
	if err := os.Chdir(withHost); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"import", "--with-host", snap}); code != 0 {
		t.Fatal("import --with-host")
	}
	if _, err := os.Stat(filepath.Join(withHost, ".opencode", "agents", "lead.md")); err != nil {
		t.Fatal("import --with-host should write .opencode/agents/lead.md")
	}
}

func TestRunRequiresPrompt(t *testing.T) {
	if Execute([]string{"run"}) != 2 {
		t.Fatal()
	}
}

func TestRunWatchBadOTLPProtocolExit2(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/json")
	if Execute([]string{"run", "-p", "hi"}) != 2 {
		t.Fatal("run bad protocol should be 2 before work")
	}
	if Execute([]string{"watch", "--once"}) != 2 {
		t.Fatal("watch bad protocol should be 2 before work")
	}
}

func TestTracesCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg"))
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	origFollow := tracesFollowFn
	tracesFollowFn = func(string) int { return 0 }
	t.Cleanup(func() { tracesFollowFn = origFollow })
	if Execute([]string{"traces", "--follow"}) != 0 {
		t.Fatal("follow flag")
	}

	if Execute([]string{"traces", "--nope"}) != 2 {
		t.Fatal("unknown flag")
	}
	if Execute([]string{"traces", "--last", "x"}) != 2 {
		t.Fatal("bad last")
	}
	if Execute([]string{"traces", "--export"}) != 2 {
		t.Fatal("export requires file")
	}

	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := traces.Append(root, traces.Span{
		Name:    "squad-oc.run",
		TraceID: "aa",
		SpanID:  "bb",
		Start:   start,
		End:     start.Add(time.Second),
		Status:  "OK",
		Attributes: map[string]string{
			"agent":        "squad",
			"prompt_bytes": "3",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := traces.Append(root, traces.Span{
		Name:       "squad-oc.watch.execute",
		TraceID:    "cc",
		SpanID:     "dd",
		Start:      start.Add(time.Minute),
		End:        start.Add(time.Minute + 2*time.Second),
		Status:     "ERROR",
		Attributes: map[string]string{"issues": "2"},
	}); err != nil {
		t.Fatal(err)
	}

	table := captureStdout(t, func() {
		if Execute([]string{"traces"}) != 0 {
			t.Fatal("traces")
		}
	})
	if !strings.Contains(table, "squad-oc.run") || !strings.Contains(table, "squad-oc.watch.execute") {
		t.Fatalf("table: %s", table)
	}

	one := captureStdout(t, func() {
		if Execute([]string{"traces", "--last", "1", "--json"}) != 0 {
			t.Fatal("json")
		}
	})
	var listed []traces.Span
	if err := json.Unmarshal([]byte(one), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "squad-oc.watch.execute" {
		t.Fatalf("json last 1: %s", one)
	}

	dest := filepath.Join(root, "out.otlp.json")
	exported := captureStdout(t, func() {
		if Execute([]string{"traces", "--export", dest}) != 0 {
			t.Fatal("export")
		}
	})
	if !strings.Contains(exported, dest) {
		t.Fatalf("export msg: %s", exported)
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"resourceSpans"`) || !strings.Contains(string(body), "squad-oc.watch.execute") {
		t.Fatalf("otlp: %s", body)
	}
}

func TestTracesCLIIngestThenExport(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg"))
	dbPath := filepath.Join(root, "oc.db")
	writeCLIIngestDB(t, dbPath, root)
	t.Setenv("OPENCODE_DB", dbPath)

	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	dest := filepath.Join(root, "ingest.otlp.json")
	out := captureStdout(t, func() {
		if Execute([]string{"traces", "--export", dest}) != 0 {
			t.Fatal("traces --export after ingest")
		}
	})
	if !strings.Contains(out, dest) {
		t.Fatalf("export msg: %s", out)
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{`"resourceSpans"`, traces.NameSession, traces.NameChat, "session.id", "gen_ai.conversation.id"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
	if strings.Contains(s, "gen_ai.input.messages") || strings.Contains(s, "SECRET") {
		t.Fatal("export leaked bodies/messages")
	}
	spans, err := traces.List(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	var chats int
	for _, sp := range spans {
		if sp.Name == traces.NameChat && sp.MessageID != "" && sp.SessionID == "ses_cli" {
			chats++
		}
	}
	if chats != 1 {
		t.Fatalf("ingested chats=%d spans=%d", chats, len(spans))
	}
}

func writeCLIIngestDB(t *testing.T, path, projectRoot string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE session (
  id text, parent_id text, directory text, agent text, model text,
  time_created integer, time_updated integer, time_archived integer
);
CREATE TABLE message (
  id text, session_id text, time_created integer, time_updated integer, data text
);
CREATE TABLE part (
  id text, message_id text, session_id text, time_created integer, time_updated integer, data text
);
INSERT INTO session VALUES ('ses_cli', NULL, ?, 'squad', '{"id":"m","providerID":"p"}', 1, 2, NULL);
INSERT INTO message VALUES
 ('u1', 'ses_cli', 1000, 1000, '{"role":"user","time":{"created":1000}}'),
 ('msg_cli', 'ses_cli', 1100, 1200, '{"role":"assistant","parentID":"u1","mode":"squad","modelID":"m","providerID":"p","cost":0.1,"tokens":{"input":1,"output":2,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":1100,"completed":1200}}');
INSERT INTO part VALUES
 ('p1', 'u1', 'ses_cli', 1000, 1000, '{"type":"text","text":"hi"}'),
 ('p2', 'msg_cli', 'ses_cli', 1100, 1200, '{"type":"text","text":"yo"}');
`, projectRoot)
	if err != nil {
		t.Fatal(err)
	}
}

type stubFailTickets struct{}

func (stubFailTickets) ListOpen(context.Context) ([]brief.Ticket, error) {
	return nil, fmt.Errorf("stubbed-brief-source")
}

type stubFailPRs struct{}

func (stubFailPRs) ListOpen(context.Context) ([]brief.PR, error) {
	return nil, fmt.Errorf("stubbed-brief-source")
}

func (stubFailPRs) ListMerged(context.Context, int) ([]brief.PR, error) {
	return nil, fmt.Errorf("stubbed-brief-source")
}

func TestBriefCLI(t *testing.T) {
	prevSources := newBriefSources
	t.Cleanup(func() { newBriefSources = prevSources })
	newBriefSources = func(string) (brief.TicketSource, brief.PRSource) {
		return stubFailTickets{}, stubFailPRs{}
	}

	if Execute([]string{"brief", "--nope"}) != 2 {
		t.Fatal("unknown flag")
	}
	empty := t.TempDir()
	prevWD, _ := os.Getwd()
	if err := os.Chdir(empty); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if Execute([]string{"brief"}) != 1 {
		t.Fatal("not initialized")
	}

	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if Execute([]string{"init", "--preset", "default", "--description", "brief"}) != 0 {
		t.Fatal("init")
	}
	out := captureStdout(t, func() {
		if Execute([]string{"brief"}) != 0 {
			t.Fatal("brief")
		}
	})
	if !strings.Contains(out, "Morning brief") || !strings.Contains(out, "unavailable") {
		t.Fatalf("%s", out)
	}
	js := captureStdout(t, func() {
		if Execute([]string{"brief", "--json"}) != 0 {
			t.Fatal("json")
		}
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(js), &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["tickets"]; !ok {
		t.Fatalf("%s", js)
	}
	if !strings.Contains(js, "stubbed-brief-source") {
		t.Fatalf("want injected stub, not live gh: %s", js)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "commands", "squad-brief.md")); err != nil {
		t.Fatal("squad-brief command missing after init")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestInitUpgradeGlobalViaCLI(t *testing.T) {
	home := isolateHome(t)
	cwd := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--global", "--description", "personal"}); code != 0 {
		t.Fatal("init --global")
	}
	root := filepath.Join(home, ".squad-oc", "global")
	team := filepath.Join(root, ".squad", "team.md")
	b, err := os.ReadFile(team)
	if err != nil || !strings.Contains(string(b), "personal") {
		t.Fatalf("global team.md: %v %s", err, b)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".squad", "config.json")); !os.IsNotExist(err) {
		t.Fatal("init --global must not write into cwd")
	}

	agent := filepath.Join(root, ".opencode", "agents", "squad.md")
	if err := os.WriteFile(agent, []byte("MUTATED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"upgrade", "--global"}); code != 0 {
		t.Fatal("upgrade --global")
	}
	got, err := os.ReadFile(agent)
	if err != nil || string(got) == "MUTATED\n" || !strings.Contains(string(got), "You are **Squad**") {
		t.Fatalf("upgrade --global did not restore host: %v %s", err, got)
	}
	after, _ := os.ReadFile(team)
	if string(after) != string(b) {
		t.Fatal("upgrade --global must leave team.md")
	}
}

func TestUpgradeGlobalRequiresInit(t *testing.T) {
	isolateHome(t)
	if Execute([]string{"upgrade", "--global"}) == 0 {
		t.Fatal("upgrade --global without init should fail")
	}
}

func TestCastRemoveViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "cli"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"cast", "--remove", "Tester"}); code != 0 {
		t.Fatal("remove")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "tester.md")); !os.IsNotExist(err) {
		t.Fatal("host agent should be gone")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "tester", "charter.md")); err != nil {
		t.Fatal("charter should remain")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "squad.md")); err != nil {
		t.Fatal("squad.md should remain")
	}
	if Execute([]string{"cast", "--remove", "Nobody"}) == 0 {
		t.Fatal("missing member should fail")
	}
	if Execute([]string{"cast", "--add", "X", "--remove", "Y"}) != 2 {
		t.Fatal("add and remove together should fail")
	}
}

func TestCastThemeOfficeAndNone(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "theme"}); code != 0 {
		t.Fatal("init")
	}
	if Execute([]string{"cast", "--theme", "parks"}) != 2 {
		t.Fatal("unknown theme should be 2")
	}
	if Execute([]string{"cast", "--theme"}) != 2 {
		t.Fatal("missing theme value should be 2")
	}
	if code := Execute([]string{"cast", "--theme", "office"}); code != 0 {
		t.Fatal("theme office")
	}
	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(team), "| Michael | Lead |") {
		t.Fatalf("office names:\n%s", team)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); err != nil {
		t.Fatal("michael.md must exist after office")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "lead.md")); !os.IsNotExist(err) {
		t.Fatal("lead.md must not exist after office")
	}
	if code := Execute([]string{"cast", "--theme", "none"}); code != 0 {
		t.Fatal("theme none")
	}
	restored, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restored), "| Lead | Lead |") || strings.Contains(string(restored), "| Michael | Lead |") {
		t.Fatalf("restored names:\n%s", restored)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "lead.md")); err != nil {
		t.Fatal("lead.md must exist after none")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); !os.IsNotExist(err) {
		t.Fatal("michael.md must be gone after none")
	}
}

func TestCastThemeNoneAfterInitOffice(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--theme", "office", "--description", "birth-none"}); code != 0 {
		t.Fatal("init --theme office")
	}
	if code := Execute([]string{"cast", "--theme", "none"}); code != 0 {
		t.Fatal("theme none after init")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "lead", "charter.md")); err != nil {
		t.Fatal("memory id must be lead again")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "michael")); !os.IsNotExist(err) {
		t.Fatal("agents/michael must be gone")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "lead.md")); err != nil {
		t.Fatal("lead.md must exist")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); !os.IsNotExist(err) {
		t.Fatal("michael.md must be gone")
	}
	cfg := squad.Detect(root).Config
	if cfg.Theme != "" || cfg.ThemeOrigin != "" {
		t.Fatalf("%+v", cfg)
	}
	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(team), "@lead") || strings.Contains(string(team), "@michael") {
		t.Fatalf("How to work:\n%s", team)
	}
}

func TestCastModelFlags(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if Execute([]string{"init", "--preset", "default", "--description", "models"}) != 0 {
		t.Fatal("init")
	}
	if Execute([]string{"cast", "--model", "squad", "xai/grok-3"}) != 0 {
		t.Fatal("set squad")
	}
	if Execute([]string{"cast", "--add", "Designer", "--role", "Design", "--model", "opencode/gpt-5.1-codex"}) != 0 {
		t.Fatal("add")
	}
	got, _ := os.ReadFile(filepath.Join(root, ".opencode", "agents", "designer.md"))
	if !strings.Contains(string(got), "model: opencode/gpt-5.1-codex") {
		t.Fatal(string(got))
	}
	if Execute([]string{"cast", "--model", "Designer", "-"}) != 0 {
		t.Fatal("clear")
	}
	got, _ = os.ReadFile(filepath.Join(root, ".opencode", "agents", "designer.md"))
	if !strings.Contains(string(got), "model: xai/grok-3") {
		t.Fatalf("should inherit squad after clear:\n%s", got)
	}
	if Execute([]string{"cast", "--model", "Lead", "nonesuch"}) != 2 {
		t.Fatal("missing slash must be usage 2")
	}
	if Execute([]string{"cast", "--model"}) != 2 {
		t.Fatal("missing args")
	}
	if Execute([]string{"cast", "--theme", "office", "--model", "Lead", "xai/grok-3"}) != 2 {
		t.Fatal("exclusive")
	}
	if Execute([]string{"cast", "--remove", "Designer", "--model", "Lead", "xai/grok-3"}) != 2 {
		t.Fatal("exclusive remove")
	}
}

func TestRecastListsHostAgentIDs(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"cast", "--theme", "office"}); code != 0 {
		t.Fatal("theme office")
	}
	out := captureStdout(t, func() {
		if Execute([]string{"recast"}) != 0 {
			t.Fatal("recast")
		}
	})
	if !strings.Contains(out, ".opencode/agents/michael.md") {
		t.Fatalf("recast should list host slug:\n%s", out)
	}
	if strings.Contains(out, ".opencode/agents/lead.md") {
		t.Fatalf("recast must not list memory id lead.md:\n%s", out)
	}
}

func TestInitThemeOfficeAndUnknown(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if Execute([]string{"init", "--theme", "parks"}) != 2 {
		t.Fatal("unknown theme should be 2")
	}
	if Execute([]string{"init", "--theme"}) != 2 {
		t.Fatal("missing theme value should be 2")
	}
	if code := Execute([]string{"init", "--theme", "office", "--description", "office-birth"}); code != 0 {
		t.Fatal("init --theme office")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "michael", "charter.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "lead.md")); !os.IsNotExist(err) {
		t.Fatal("no dual files")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "mentions.md")); !os.IsNotExist(err) {
		t.Fatal("birth theme writes no mention map")
	}
	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(team), "@michael") {
		t.Fatalf("How to work:\n%s", team)
	}
	help := captureStdout(t, func() {
		if Execute([]string{"help"}) != 0 {
			t.Fatal("help")
		}
	})
	if !strings.Contains(help, "[--theme office|none]") {
		t.Fatalf("help missing init --theme: %s", help)
	}
}

func TestMCPUnknownAndMissingArgs(t *testing.T) {
	if Execute([]string{"mcp"}) != 2 {
		t.Fatal("mcp without subcommand should be 2")
	}
	if Execute([]string{"mcp", "nope"}) != 2 {
		t.Fatal("unknown mcp subcommand should be 2")
	}
}

func TestMCPInitApplyListViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "mcp"}); code != 0 {
		t.Fatal("init")
	}
	help := captureStdout(t, func() {
		if Execute([]string{"help"}) != 0 {
			t.Fatal("help")
		}
	})
	if !strings.Contains(help, "mcp apply") {
		t.Fatalf("help missing mcp line: %s", help)
	}
	if code := Execute([]string{"mcp", "init"}); code != 0 {
		t.Fatal("mcp init")
	}
	cfg := filepath.Join(root, ".squad", "mcp-config.json")
	if _, err := os.Stat(cfg); err != nil {
		t.Fatal(err)
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" }
    }
  }
}`)
	if err := os.WriteFile(cfg, org, 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"mcp", "apply"}); code != 0 {
		t.Fatal("mcp apply")
	}
	raw, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"github"`) || !strings.Contains(string(raw), `"$schema"`) {
		t.Fatalf("apply result: %s", raw)
	}
	if strings.Contains(string(raw), "sk-") || strings.Contains(string(raw), "ghp_") {
		t.Fatalf("apply wrote a token: %s", raw)
	}
	listed := captureStdout(t, func() {
		if Execute([]string{"mcp", "list"}) != 0 {
			t.Fatal("mcp list")
		}
	})
	if !strings.Contains(listed, "github") {
		t.Fatalf("list: %s", listed)
	}
	if strings.Contains(listed, "sk-") || strings.Contains(listed, "ghp_") || strings.Contains(listed, "${GITHUB_TOKEN}") {
		t.Fatalf("list leaked secrets: %s", listed)
	}
}

func TestMCPApplyReadsLinkedTeamViaCLI(t *testing.T) {
	service := t.TempDir()
	shared := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := os.Chdir(shared); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"init", "--preset", "default", "--description", "shared"}); code != 0 {
		t.Fatal("init shared")
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "TOKEN": "${GITHUB_TOKEN}" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(shared, ".squad", "mcp-config.json"), org, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(service); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"init", "--preset", "default", "--description", "svc"}); code != 0 {
		t.Fatal("init service")
	}
	if code := Execute([]string{"link", shared}); code != 0 {
		t.Fatal("link")
	}
	if code := Execute([]string{"mcp", "apply"}); code != 0 {
		t.Fatal("mcp apply")
	}
	raw, err := os.ReadFile(filepath.Join(service, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"github"`) {
		t.Fatalf("linked apply: %s", raw)
	}
}

func TestMarketplaceUnknownAndMissingArgs(t *testing.T) {
	if Execute([]string{"marketplace"}) != 2 {
		t.Fatal("marketplace without subcommand should be 2")
	}
	if Execute([]string{"marketplace", "nope"}) != 2 {
		t.Fatal("unknown marketplace subcommand should be 2")
	}
	if Execute([]string{"marketplace", "add"}) != 2 {
		t.Fatal("marketplace add without args should be 2")
	}
	if Execute([]string{"marketplace", "install"}) != 2 {
		t.Fatal("marketplace install without plugin should be 2")
	}
}

func TestMarketplaceBrowseInstallViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "mkt"}); code != 0 {
		t.Fatal("init")
	}
	help := captureStdout(t, func() {
		if Execute([]string{"help"}) != 0 {
			t.Fatal("help")
		}
	})
	if !strings.Contains(help, "marketplace add") {
		t.Fatalf("help missing marketplace line: %s", help)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	pack := filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "workshop", "fixtures", "skills-pack")
	if code := Execute([]string{"marketplace", "add", "community", pack}); code != 0 {
		t.Fatal("add")
	}
	listed := captureStdout(t, func() {
		if Execute([]string{"marketplace", "list"}) != 0 {
			t.Fatal("list")
		}
	})
	if !strings.Contains(listed, "community") {
		t.Fatalf("list: %s", listed)
	}
	browsed := captureStdout(t, func() {
		if Execute([]string{"marketplace", "browse"}) != 0 {
			t.Fatal("browse")
		}
	})
	if !strings.Contains(browsed, "reflect") || !strings.Contains(browsed, "fact-checking") {
		t.Fatalf("browse: %s", browsed)
	}
	if !strings.Contains(browsed, "retrospective") {
		t.Fatalf("browse missing triggers: %s", browsed)
	}
	if code := Execute([]string{"marketplace", "install", "reflect"}); code != 0 {
		t.Fatal("install")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"marketplace", "install", "reflect", "--from", "community"}); code != 0 {
		t.Fatal("second install")
	}
}

func TestPluginInstallListUninstallViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "plug"}); code != 0 {
		t.Fatal("init")
	}
	help := captureStdout(t, func() {
		if Execute([]string{"help"}) != 0 {
			t.Fatal("help")
		}
	})
	if !strings.Contains(help, "plugin install <name>@<marketplace>") {
		t.Fatalf("help missing plugin line: %s", help)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	pack := filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "workshop", "fixtures", "skills-pack")
	if code := Execute([]string{"marketplace", "add", "community", pack}); code != 0 {
		t.Fatal("add")
	}

	if Execute([]string{"plugin"}) != 2 {
		t.Fatal("plugin without subcommand should be 2")
	}
	if Execute([]string{"plugin", "install"}) != 2 {
		t.Fatal("plugin install without spec should be 2")
	}
	if Execute([]string{"plugin", "uninstall"}) != 2 {
		t.Fatal("plugin uninstall without name should be 2")
	}

	if code := Execute([]string{"plugin", "install", "reflect@community"}); code != 0 {
		t.Fatal("plugin install")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	listed := captureStdout(t, func() {
		if Execute([]string{"plugin", "list"}) != 0 {
			t.Fatal("list")
		}
	})
	if !strings.Contains(listed, "reflect") || !strings.Contains(listed, "community") || !strings.Contains(listed, "unknown") {
		t.Fatalf("plugin list: %s", listed)
	}

	if code := Execute([]string{"plugin", "uninstall", "reflect"}); code != 0 {
		t.Fatal("uninstall")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "reflect")); !os.IsNotExist(err) {
		t.Fatalf("skill still present: %v", err)
	}

	if code := Execute([]string{"marketplace", "install", "reflect@community"}); code != 0 {
		t.Fatal("marketplace install alias")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestLinkGitURLViaCLI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	remote := makeCLITeamRemote(t, "cli-remote")

	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "svc"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"link", remote}); code != 0 {
		t.Fatal("link url")
	}
	if squad.Detect(root).Config.LinkURL != remote {
		t.Fatal("config")
	}
	if code := Execute([]string{"status"}); code != 0 {
		t.Fatal("status")
	}
	if code := Execute([]string{"link", "--sync"}); code != 0 {
		t.Fatal("sync")
	}
	if code := Execute([]string{"link", "--off"}); code != 0 {
		t.Fatal("off")
	}
	if squad.Detect(root).Config.LinkPath != "" {
		t.Fatal("still linked")
	}
	if code := Execute([]string{"link", remote}); code != 0 {
		t.Fatal("re-link")
	}
}

func TestLinkSyncWithoutRemoteIsError(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if code := Execute([]string{"init", "--preset", "default"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"link", "--sync"}); code == 0 {
		t.Fatal("expected error")
	}
}

func makeCLITeamRemote(t *testing.T, desc string) string {
	t.Helper()
	work := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: work, ProjectDescription: desc}); err != nil {
		t.Fatal(err)
	}
	runCLIGit(t, work, "init")
	runCLIGit(t, work, "config", "user.email", "test@example.com")
	runCLIGit(t, work, "config", "user.name", "test")
	runCLIGit(t, work, "add", ".")
	runCLIGit(t, work, "commit", "-m", "team")
	remote := filepath.Join(t.TempDir(), "platform.git")
	runCLIGit(t, work, "clone", "--bare", work, remote)
	return remote
}

func runCLIGit(t *testing.T, dir string, args ...string) {
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
