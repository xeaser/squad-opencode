package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/traces"
)

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

func TestTracesCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

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
