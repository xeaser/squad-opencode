package share

import (
	"os"
	"path/filepath"
	"runtime"
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

func workshopSkillsPack(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	p := filepath.Join(filepath.Dir(file), "..", "..", "docs", "workshop", "fixtures", "skills-pack")
	if !isDir(p) {
		t.Fatalf("missing fixture pack %s", p)
	}
	return p
}

func TestMarketplaceBrowseAndInstall(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root, ProjectDescription: "keep-me"}); err != nil {
		t.Fatal(err)
	}
	pack := workshopSkillsPack(t)
	if err := AddMarketplace(root, "community", pack); err != nil {
		t.Fatal(err)
	}
	list, err := ListMarketplaces(root)
	if err != nil || len(list) != 1 || list[0].Name != "community" {
		t.Fatalf("list: %v %v", list, err)
	}

	plugins, err := BrowsePlugins(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 2 {
		t.Fatalf("browse want 2 plugins, got %#v", plugins)
	}
	byName := map[string]Plugin{}
	for _, p := range plugins {
		byName[p.Name] = p
	}
	if _, ok := byName["reflect"]; !ok {
		t.Fatalf("missing reflect: %#v", plugins)
	}
	if _, ok := byName["fact-checking"]; !ok {
		t.Fatalf("missing fact-checking: %#v", plugins)
	}
	if !strings.Contains(byName["reflect"].Triggers, "retrospective") {
		t.Fatalf("reflect triggers: %q", byName["reflect"].Triggers)
	}
	named, err := BrowsePlugins(root, "community")
	if err != nil || len(named) != 2 {
		t.Fatalf("browse community: %v %#v", err, named)
	}

	n, err := InstallPlugin(root, "reflect", "")
	if err != nil || n < 1 {
		t.Fatalf("install n=%d err=%v", n, err)
	}
	skill := filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")
	got, err := os.ReadFile(skill)
	if err != nil || !strings.Contains(string(got), "retrospective") {
		t.Fatalf("copied SKILL.md: %v %s", err, got)
	}

	n2, err := InstallPlugin(root, "reflect", "community")
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	again, err := os.ReadFile(skill)
	if err != nil || string(again) != string(got) {
		t.Fatalf("idempotent install changed SKILL.md: n=%d %v %s", n2, err, again)
	}

	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil || !strings.Contains(string(team), "keep-me") {
		t.Fatalf("team.md touched: %v %s", err, team)
	}
	decisions := filepath.Join(root, ".squad", "decisions.md")
	beforeDec, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}

	poison := t.TempDir()
	if _, err := squad.CopyTree(pack, poison); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(poison, ".squad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(poison, ".squad", "team.md"), []byte("SHOULD-NOT-WIN"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(poison, ".squad", "decisions.md"), []byte("SHOULD-NOT-WIN"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddMarketplace(root, "poison", poison); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPlugin(root, "fact-checking", "poison"); err != nil {
		t.Fatal(err)
	}
	team, err = os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil || strings.Contains(string(team), "SHOULD-NOT-WIN") || !strings.Contains(string(team), "keep-me") {
		t.Fatalf("install overwrote team.md: %v %s", err, team)
	}
	afterDec, err := os.ReadFile(decisions)
	if err != nil || string(afterDec) != string(beforeDec) || strings.Contains(string(afterDec), "SHOULD-NOT-WIN") {
		t.Fatalf("install overwrote decisions.md: %v %s", err, afterDec)
	}

	if _, err := BrowsePlugins(root, "nope"); err == nil || !strings.Contains(err.Error(), "unknown marketplace") {
		t.Fatalf("unknown marketplace: %v", err)
	}
	if _, err := InstallPlugin(root, "nope", "community"); err == nil || !strings.Contains(err.Error(), "missing plugin") {
		t.Fatalf("missing plugin: %v", err)
	}
	if _, err := InstallPlugin(root, "reflect", "missing"); err == nil || !strings.Contains(err.Error(), "missing marketplace") {
		t.Fatalf("missing marketplace: %v", err)
	}

	if err := RemoveMarketplace(root, "poison"); err != nil {
		t.Fatal(err)
	}
	list, _ = ListMarketplaces(root)
	if len(list) != 1 || list[0].Name != "community" {
		t.Fatal(list)
	}
}

func TestMarketplaceGitURLUsesCloneGit(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := workshopSkillsPack(t)
	prev := CloneGit
	var urls []string
	CloneGit = func(url, dest string) error {
		urls = append(urls, url)
		if url != "https://example.com/skills.git" {
			t.Fatalf("url %s", url)
		}
		_, err := squad.CopyTree(pack, dest)
		return err
	}
	t.Cleanup(func() { CloneGit = prev })

	if err := AddMarketplace(root, "community", "https://example.com/skills.git"); err != nil {
		t.Fatal(err)
	}
	plugins, err := BrowsePlugins(root, "community")
	if err != nil || len(plugins) != 2 {
		t.Fatalf("browse git: %v %#v", err, plugins)
	}
	n, err := InstallPlugin(root, "reflect", "community")
	if err != nil || n < 1 {
		t.Fatalf("install git n=%d err=%v", n, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if len(urls) < 2 {
		t.Fatalf("CloneGit calls: %v", urls)
	}
}

func TestMarketplaceInstallRequiresFromWhenMany(t *testing.T) {
	root := t.TempDir()
	pack := workshopSkillsPack(t)
	if err := AddMarketplace(root, "a", pack); err != nil {
		t.Fatal(err)
	}
	if err := AddMarketplace(root, "b", pack); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPlugin(root, "reflect", ""); err == nil || !strings.Contains(err.Error(), "--from") {
		t.Fatalf("expected --from required: %v", err)
	}
}

func TestNamedPluginInstallListUninstall(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root, ProjectDescription: "keep-me"}); err != nil {
		t.Fatal(err)
	}
	pack := workshopSkillsPack(t)
	if err := AddMarketplace(root, "community", pack); err != nil {
		t.Fatal(err)
	}

	n, err := InstallPlugin(root, "reflect@community", "")
	if err != nil || n < 1 {
		t.Fatalf("install name@source n=%d err=%v", n, err)
	}
	skill := filepath.Join(root, ".opencode", "skills", "reflect", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatal(err)
	}

	listed, err := ListInstalledPlugins(root)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v %#v", err, listed)
	}
	if listed[0].Name != "reflect" || listed[0].Source != "community" || listed[0].Version != "unknown" {
		t.Fatalf("record: %#v", listed[0])
	}
	if listed[0].InstalledAt == "" {
		t.Fatal("installedAt empty")
	}

	versPack := t.TempDir()
	plugDir := filepath.Join(versPack, "plugins", "reflect")
	if err := os.MkdirAll(plugDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugDir, "SKILL.md"), []byte("# v\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugDir, "manifest.json"), []byte(`{"version":"1.2.3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddMarketplace(root, "versioned", versPack); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPlugin(root, "reflect@versioned", ""); err != nil {
		t.Fatal(err)
	}
	listed, err = ListInstalledPlugins(root)
	if err != nil || len(listed) != 1 || listed[0].Version != "1.2.3" || listed[0].Source != "versioned" {
		t.Fatalf("versioned: %v %#v", err, listed)
	}

	missMkt, err := InstallPlugin(root, "reflect@no-such-market", "")
	if err == nil || !strings.Contains(err.Error(), "missing marketplace") || strings.Contains(err.Error(), "missing plugin") {
		t.Fatalf("missing marketplace: n=%d err=%v", missMkt, err)
	}
	missPlug, err := InstallPlugin(root, "no-such@community", "")
	if err == nil || !strings.Contains(err.Error(), "missing plugin") || strings.Contains(err.Error(), "missing marketplace") {
		t.Fatalf("missing plugin: n=%d err=%v", missPlug, err)
	}

	charter := filepath.Join(root, ".squad", "agents", "lead", "charter.md")
	decisions := filepath.Join(root, ".squad", "decisions.md")
	team := filepath.Join(root, ".squad", "team.md")
	beforeCharter, err := os.ReadFile(charter)
	if err != nil {
		t.Fatal(err)
	}
	beforeDec, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}
	beforeTeam, err := os.ReadFile(team)
	if err != nil {
		t.Fatal(err)
	}

	if err := UninstallPlugin(root, "reflect"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(skill); !os.IsNotExist(err) {
		t.Fatalf("skill still present: %v", err)
	}
	afterCharter, err := os.ReadFile(charter)
	if err != nil || string(afterCharter) != string(beforeCharter) {
		t.Fatalf("charter touched: %v %s", err, afterCharter)
	}
	afterDec, err := os.ReadFile(decisions)
	if err != nil || string(afterDec) != string(beforeDec) {
		t.Fatalf("decisions touched: %v %s", err, afterDec)
	}
	afterTeam, err := os.ReadFile(team)
	if err != nil || string(afterTeam) != string(beforeTeam) || !strings.Contains(string(afterTeam), "keep-me") {
		t.Fatalf("team touched: %v %s", err, afterTeam)
	}
	listed, err = ListInstalledPlugins(root)
	if err != nil || len(listed) != 0 {
		t.Fatalf("list after uninstall: %v %#v", err, listed)
	}

	if _, err := InstallPlugin(root, "reflect@community", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	listed, err = ListInstalledPlugins(root)
	if err != nil || len(listed) != 1 || listed[0].Source != "community" {
		t.Fatalf("reinstall list: %v %#v", err, listed)
	}
}
