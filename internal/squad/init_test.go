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

	// idempotent
	res2, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Preset: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if !res2.AlreadyInitialized || len(res2.FilesWritten) != 0 {
		t.Fatalf("expected idempotent skip, got %+v", res2)
	}
	if !IsInitialized(root) {
		t.Error("expected initialized")
	}
	d := Detect(root)
	if !d.Initialized || d.Config == nil || d.Config.Host != "opencode" {
		t.Errorf("detect: %+v", d)
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

