package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
