package share

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestUpstreamAndPack(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	agentDir := filepath.Join(pack, ".opencode", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "extra.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddUpstream(root, "local", pack); err != nil {
		t.Fatal(err)
	}
	list, err := ListUpstreams(root)
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
	n, err := SyncUpstream(root, "local")
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "extra.md")); err != nil {
		t.Fatal(err)
	}
	n, err = InstallPack(root, pack)
	if err != nil || n < 1 {
		t.Fatal(n, err)
	}
	if err := RemoveUpstream(root, "local"); err != nil {
		t.Fatal(err)
	}
	list, _ = ListUpstreams(root)
	if len(list) != 0 {
		t.Fatal(list)
	}
}

func TestPackAddsNewAgentWithoutTouchingTeam(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root, ProjectDescription: "keep-me"}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	newAgent := filepath.Join(pack, ".squad", "agents", "designer")
	if err := os.MkdirAll(newAgent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newAgent, "charter.md"), []byte("design"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, ".squad", "team.md"), []byte("SHOULD-NOT-WIN"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := InstallPack(root, pack)
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".squad", "agents", "designer", "charter.md"))
	if err != nil || string(got) != "design" {
		t.Fatalf("%v %s", err, got)
	}
	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil || !strings.Contains(string(team), "keep-me") {
		t.Fatalf("team overwritten: %v %s", err, team)
	}
	if strings.Contains(string(team), "SHOULD-NOT-WIN") {
		t.Fatal("pack overwrote team.md")
	}
}

func TestPackBareHostTree(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pack, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "agents", "reviewer.md"), []byte("rev"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(root, pack); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "reviewer.md")); err != nil {
		t.Fatal(err)
	}
}

func TestApplySourceGitURL(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	cloned := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cloned, ".opencode", "skills", "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloned, ".opencode", "skills", "extra", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := CloneGit
	CloneGit = func(url, dest string) error {
		if url != "https://example.com/pack.git" {
			t.Fatalf("url %s", url)
		}
		_, err := squad.CopyTree(cloned, dest)
		return err
	}
	t.Cleanup(func() { CloneGit = prev })

	n, err := ApplySource(root, "https://example.com/pack.git")
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "extra", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestApplySourceMissing(t *testing.T) {
	if _, err := ApplySource(t.TempDir(), filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error")
	}
}

func TestLink(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err := Link(root, other); err != nil {
		t.Fatal(err)
	}
	if squad.Detect(root).Config.LinkPath == "" {
		t.Fatal("expected link")
	}
}
