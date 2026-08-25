package squad

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDefaultPreset(t *testing.T) {
	root := t.TempDir()
	res, err := WriteDefaultPreset(InitOptions{
		ProjectRoot:        root,
		Preset:             "default",
		ProjectDescription: "Recipe app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AlreadyInitialized {
		t.Fatal("expected first init")
	}
	if len(res.FilesWritten) < 10 {
		t.Fatalf("expected many files, got %d", len(res.FilesWritten))
	}

	required := []string{
		".squad/config.json",
		".squad/.gitignore",
		".squad/comms/.gitkeep",
		".squad/team.md",
		".squad/charter.md",
		".squad/decisions.md",
		".squad/ceremonies.md",
		".squad/agents/lead/charter.md",
		".squad/agents/frontend/charter.md",
		".squad/agents/backend/charter.md",
		".squad/agents/tester/charter.md",
		".opencode/agents/squad.md",
		".opencode/agents/lead.md",
		".opencode/agents/frontend.md",
		".opencode/agents/backend.md",
		".opencode/agents/tester.md",
		".opencode/skills/squad-team/SKILL.md",
		".opencode/skills/squad-handoff/SKILL.md",
		".opencode/commands/squad-status.md",
		".opencode/commands/squad-cast.md",
		".opencode/commands/squad-review.md",
		".opencode/.gitignore",
		"opencode.json",
	}
	for _, rel := range required {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}

	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(team), "Recipe app") {
		t.Error("team.md missing description")
	}

	var cfg Config
	data, _ := os.ReadFile(ConfigPath(root))
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "opencode" || cfg.Preset != "default" || cfg.ProjectDescription != "Recipe app" {
		t.Errorf("bad config: %+v", cfg)
	}

	cerPath := filepath.Join(root, ".squad", "ceremonies.md")
	cer, err := os.ReadFile(cerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cer), "Design Review") || !strings.Contains(string(cer), "Retro") {
		t.Error("ceremonies.md missing Design Review / Retro")
	}
	customCer := "# Ceremonies\n\nCUSTOM TEAM EDIT\n"
	if err := os.WriteFile(cerPath, []byte(customCer), 0o644); err != nil {
		t.Fatal(err)
	}

	// idempotent
	res2, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Preset: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if !res2.AlreadyInitialized || len(res2.FilesWritten) != 0 {
		t.Fatalf("expected idempotent skip, got %+v", res2)
	}
	gotCer, err := os.ReadFile(cerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotCer) != customCer {
		t.Fatal("second init must not overwrite ceremonies.md")
	}
	if !IsInitialized(root) {
		t.Error("expected initialized")
	}
	d := Detect(root)
	if !d.Initialized || d.Config == nil || d.Config.Host != "opencode" {
		t.Errorf("detect: %+v", d)
	}
}

func TestInitThemeOfficeNativeIDs(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Theme: ThemeOffice}); err != nil {
		t.Fatal(err)
	}
	cfg := Detect(root).Config
	if cfg.Theme != ThemeOffice || cfg.ThemeOrigin != ThemeOriginInit {
		t.Fatalf("%+v", cfg)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "michael", "charter.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "lead")); !os.IsNotExist(err) {
		t.Fatal("birth office must not keep agents/lead")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "lead.md")); !os.IsNotExist(err) {
		t.Fatal("no dual files")
	}
	if _, err := os.Stat(MentionsPath(root)); !os.IsNotExist(err) {
		t.Fatal("birth theme writes no mention map")
	}
	team, _ := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if !strings.Contains(string(team), "@michael") {
		t.Fatal("How to work should document @michael")
	}
}

func TestParseTeamMarkdown(t *testing.T) {
	md := `
## Coordinator

| Name | Role | Notes |
|------|------|-------|
| Squad | Coordinator | Routes work |

## Members

| Name | Role | Charter | Status |
|------|------|---------|--------|
| Lead | Lead | x | Active |
| Frontend | Frontend | y | Active |
`
	members := ParseTeamMarkdown(md)
	if len(members) != 2 {
		t.Fatalf("want 2 members, got %d: %+v", len(members), members)
	}
	if members[0].Name != "Lead" || members[1].Name != "Frontend" {
		t.Fatalf("unexpected: %+v", members)
	}
}

func TestParseTeamMarkdownModelColumn(t *testing.T) {
	md := `
## Coordinator

| Name | Role | Model | Notes |
|------|------|-------|-------|
| Squad | Coordinator | xai/grok-3 | Routes work |

## Members

| Name | Role | Charter | Status | Model |
|------|------|---------|--------|-------|
| Lead | Lead | ` + "`.squad/agents/lead/charter.md`" + ` | Active | anthropic/claude-sonnet-4-5 |
| Tester | Tester | ` + "`.squad/agents/tester/charter.md`" + ` | Active | |
`
	members := ParseTeamMarkdown(md)
	if len(members) != 2 {
		t.Fatalf("want 2, got %+v", members)
	}
	if members[0].Model != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("lead model: %+v", members[0])
	}
	if members[1].Model != "" {
		t.Fatalf("tester must be empty inherit: %+v", members[1])
	}
	if got := ParseSquadModel(md); got != "xai/grok-3" {
		t.Fatalf("squad model %q", got)
	}
}

func TestParseTeamMarkdownOldFourColumn(t *testing.T) {
	md := `
## Coordinator

| Name | Role | Notes |
|------|------|-------|
| Squad | Coordinator | Routes work |

## Members

| Name | Role | Charter | Status |
|------|------|---------|--------|
| Lead | Lead | x | Active |
`
	members := ParseTeamMarkdown(md)
	if len(members) != 1 || members[0].Name != "Lead" || members[0].Status != "Active" || members[0].Model != "" {
		t.Fatalf("%+v", members)
	}
	if ParseSquadModel(md) != "" {
		t.Fatal("missing model column must be empty")
	}
}

func TestEffectiveModel(t *testing.T) {
	if EffectiveModel("anthropic/claude-sonnet-4-5", "xai/grok-3") != "anthropic/claude-sonnet-4-5" {
		t.Fatal("own wins")
	}
	if EffectiveModel("", "xai/grok-3") != "xai/grok-3" {
		t.Fatal("inherit team")
	}
	if EffectiveModel("  ", "") != "" {
		t.Fatal("both empty")
	}
}

func TestReadTeamAfterInit(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	members, err := ReadTeam(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) < 4 {
		t.Fatalf("want >=4 members, got %d", len(members))
	}
}

func TestInitTeamMarkdownHasModelColumn(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(ResolveDir(root), "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Model") {
		t.Fatalf("new team.md should include Model column:\n%s", raw)
	}
	if ParseSquadModel(string(raw)) != "" {
		t.Fatal("default model must be empty")
	}
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestWriteDefaultPresetGlobal(t *testing.T) {
	home := isolateHome(t)
	res, err := WriteDefaultPreset(InitOptions{
		Global:             true,
		ProjectDescription: "Personal notes",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := filepath.Join(home, ".squad-oc", "global")
	if res.ProjectRoot != wantRoot {
		t.Fatalf("root %s want %s", res.ProjectRoot, wantRoot)
	}
	team := filepath.Join(home, ".squad-oc", "global", ".squad", "team.md")
	data, err := os.ReadFile(team)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Personal notes") {
		t.Errorf("team.md missing description: %s", data)
	}
	if !IsInitialized(wantRoot) {
		t.Error("expected initialized")
	}
}

func TestUpgradeHostFilesGlobal(t *testing.T) {
	home := isolateHome(t)
	res, err := WriteDefaultPreset(InitOptions{Global: true, ProjectDescription: "Keep me"})
	if err != nil {
		t.Fatal(err)
	}
	root := res.ProjectRoot
	if root != filepath.Join(home, ".squad-oc", "global") {
		t.Fatalf("root %s", root)
	}

	teamPath := filepath.Join(root, ".squad", "team.md")
	teamBefore, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	cerPath := filepath.Join(root, ".squad", "ceremonies.md")
	customCer := "# Ceremonies\n\nCUSTOM GLOBAL EDIT\n"
	if err := os.WriteFile(cerPath, []byte(customCer), 0o644); err != nil {
		t.Fatal(err)
	}
	agent := filepath.Join(root, ".opencode", "agents", "squad.md")
	if err := os.WriteFile(agent, []byte("MUTATED\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	up, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(up.Updated) == 0 {
		t.Fatalf("expected squad.md updated, got %+v", up)
	}

	got, err := os.ReadFile(agent)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "MUTATED\n" || !strings.Contains(string(got), "You are **Squad**") {
		t.Fatalf("agent not restored from template:\n%s", got)
	}

	teamAfter, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(teamAfter) != string(teamBefore) {
		t.Fatal("team.md must not change on upgrade")
	}
	if !strings.Contains(string(teamAfter), "Keep me") {
		t.Fatal("description lost")
	}
	gotCer, err := os.ReadFile(cerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotCer) != customCer {
		t.Fatal("ceremonies.md must not change on upgrade")
	}
}
