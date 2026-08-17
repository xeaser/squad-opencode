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

func TestPackCopiesMissingMCPConfigOnly(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pack, ".opencode", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, ".opencode", "agents", "extra.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "mcp-config.json"), []byte(`{"mcpServers":{"from-pack":{"command":"npx","args":["-y","demo"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(root, pack); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(squad.ResolveDir(root), "mcp-config.json")
	got, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(got), "from-pack") {
		t.Fatalf("pack-root mcp not copied: %v %s", err, got)
	}
	if err := os.WriteFile(dest, []byte(`{"mcpServers":{"keep":{"command":"npx"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(root, pack); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(again), "keep") || strings.Contains(string(again), "from-pack") {
		t.Fatalf("existing org MCP overwritten: %s", again)
	}
}

func TestPackSquadMCPConfigMissingOnly(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pack, ".opencode", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, ".opencode", "agents", "extra.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pack, ".squad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, ".squad", "mcp-config.json"), []byte(`{"mcpServers":{"nested":{"command":"npx"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(root, pack); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, ".squad", "mcp-config.json")
	got, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(got), "nested") {
		t.Fatalf(".squad mcp-config not copied: %v %s", err, got)
	}
	if err := os.WriteFile(dest, []byte(`{"mcpServers":{"local":{"command":"npx"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(root, pack); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(again), "local") {
		t.Fatalf("existing .squad mcp-config overwritten: %s", again)
	}
}

func TestPackRootMCPFollowsLink(t *testing.T) {
	service := t.TempDir()
	shared := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: service}); err != nil {
		t.Fatal(err)
	}
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: shared}); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(service, shared); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pack, ".opencode", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, ".opencode", "skills", "x.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "mcp-config.json"), []byte(`{"mcpServers":{"linked":{"command":"npx"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPack(service, pack); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(squad.ResolveDir(service), "mcp-config.json")
	got, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(got), "linked") {
		t.Fatalf("pack-root mcp not in linked team: %v %s", err, got)
	}
	if _, err := os.Stat(filepath.Join(service, ".squad", "mcp-config.json")); !os.IsNotExist(err) {
		t.Fatal("pack-root mcp should not land in the local .squad when linked")
	}
}

func TestLink(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: other, ProjectDescription: "shared"}); err != nil {
		t.Fatal(err)
	}
	dest, err := Link(root, other)
	if err != nil {
		t.Fatal(err)
	}
	if dest != squad.SquadDir(other) {
		t.Fatal(dest)
	}
	if squad.Detect(root).Config.LinkPath == "" {
		t.Fatal("expected link")
	}
	if err := Unlink(root); err != nil {
		t.Fatal(err)
	}
	if squad.Detect(root).Config.LinkPath != "" {
		t.Fatal("expected unlink")
	}
}
