package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddMemberAndRecast(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := AddMember(root, "Designer", "Design", ""); err != nil {
		t.Fatal(err)
	}
	members, err := ReadTeam(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range members {
		if m.ID == "designer" && m.Role == "Design" {
			found = true
		}
	}
	if !found {
		t.Fatalf("designer missing: %+v", members)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "designer", "charter.md")); err != nil {
		t.Fatal(err)
	}

	res, err := Recast(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Written < 5 {
		t.Fatalf("expected stock + designer, got %+v", res)
	}
	got, err := os.ReadFile(filepath.Join(root, ".opencode", "agents", "designer.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Designer") || !strings.Contains(string(got), ".squad/agents/designer/charter.md") {
		t.Fatal(string(got))
	}
	// Coordinator agent is left alone.
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "squad.md")); err != nil {
		t.Fatal(err)
	}
}

func TestAddMemberRejectsDuplicate(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := AddMember(root, "Lead", "Lead", ""); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestRecastUsesStockLeadTemplate(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	lead := filepath.Join(root, ".opencode", "agents", "lead.md")
	if err := os.WriteFile(lead, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(lead)
	if !strings.Contains(string(got), "You are **Lead**") {
		t.Fatal(string(got))
	}
}

func TestRecastBirthUsesStockLeadTemplate(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Theme: ThemeOffice}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".opencode", "agents", "michael.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(got)
	if !strings.Contains(body, "You are **Lead**") {
		t.Fatalf("birth recast must use stock Lead prompt, not a generic stub:\n%s", body)
	}
	if strings.Contains(body, "You are **Michael**") {
		t.Fatalf("birth recast must not fall back to generic stub:\n%s", body)
	}
	if !strings.Contains(body, ".squad/agents/michael/") {
		t.Fatalf("birth michael.md must point at native agent dir:\n%s", body)
	}
	if strings.Contains(body, ".squad/agents/lead/") {
		t.Fatalf("birth michael.md must not keep stock role path (shadow agents/lead):\n%s", body)
	}
	for _, pair := range []struct{ host, role string }{
		{"jim", "frontend"},
		{"dwight", "backend"},
		{"pam", "tester"},
	} {
		gotRole, err := os.ReadFile(filepath.Join(root, ".opencode", "agents", pair.host+".md"))
		if err != nil {
			t.Fatal(err)
		}
		roleBody := string(gotRole)
		if !strings.Contains(roleBody, ".squad/agents/"+pair.host+"/") {
			t.Fatalf("birth %s.md must point at native agent dir:\n%s", pair.host, roleBody)
		}
		if strings.Contains(roleBody, ".squad/agents/"+pair.role+"/") {
			t.Fatalf("birth %s.md must not keep stock role path:\n%s", pair.host, roleBody)
		}
	}
}

func TestRecastAppliedKeepsStockLeadPrompt(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyTheme(root, ThemeOffice); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".opencode", "agents", "michael.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "You are **Lead**") {
		t.Fatalf("applied recast must keep full Lead prompt:\n%s", got)
	}
}

func TestRemoveMemberDeletesHostAgent(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMember(root, "Tester"); err != nil {
		t.Fatal(err)
	}
	members, err := ReadTeam(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		if m.ID == "tester" {
			t.Fatal("still present")
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "tester.md")); !os.IsNotExist(err) {
		t.Fatal("host agent should be gone")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "tester", "charter.md")); err != nil {
		t.Fatal("charter should remain")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "squad.md")); err != nil {
		t.Fatal("coordinator agent should remain")
	}
}

func TestRemoveMemberDeletesThemedHostSlug(t *testing.T) {
	t.Run("applied", func(t *testing.T) {
		root := t.TempDir()
		if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
			t.Fatal(err)
		}
		if err := ApplyTheme(root, ThemeOffice); err != nil {
			t.Fatal(err)
		}
		if _, err := Recast(root); err != nil {
			t.Fatal(err)
		}
		if err := RemoveMember(root, "Tester"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "pam.md")); !os.IsNotExist(err) {
			t.Fatal("applied office: pam.md must be removed")
		}
		if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "tester", "charter.md")); err != nil {
			t.Fatal("applied office: tester charter should remain")
		}
	})

	t.Run("birth", func(t *testing.T) {
		root := t.TempDir()
		if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Theme: ThemeOffice}); err != nil {
			t.Fatal(err)
		}
		if err := RemoveMember(root, "Pam"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "pam.md")); !os.IsNotExist(err) {
			t.Fatal("birth: pam.md must be removed")
		}
		if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "pam", "charter.md")); err != nil {
			t.Fatal("birth: pam charter should remain")
		}
	})
}

func TestRemoveMemberMatchesIDAndName(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMember(root, "frontend"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMember(root, "BACKEND"); err != nil {
		t.Fatal(err)
	}
	members, err := ReadTeam(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		if m.ID == "frontend" || m.ID == "backend" {
			t.Fatalf("still present: %+v", m)
		}
	}
}

func TestRemoveMemberRejectsMissingAndSquad(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMember(root, "Nobody"); err == nil {
		t.Fatal("expected not-found error")
	}
	if err := RemoveMember(root, "squad"); err == nil {
		t.Fatal("expected reserved error")
	}
	if err := RemoveMember(root, "Squad"); err == nil {
		t.Fatal("expected reserved error")
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "squad.md")); err != nil {
		t.Fatal("squad.md must not be deleted")
	}
}

func TestRecastWritesMemberModelAndInheritsSquad(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	team := filepath.Join(ResolveDir(root), "team.md")
	body := `# Mission Control

## Coordinator

| Name | Role | Model | Notes |
|------|------|-------|-------|
| Squad | Coordinator | xai/grok-3 | Routes |

## Members

| Name | Role | Charter | Status | Model |
|------|------|---------|--------|-------|
| Lead | Lead | ` + "`.squad/agents/lead/charter.md`" + ` | Active | anthropic/claude-sonnet-4-5 |
| Frontend | Frontend | ` + "`.squad/agents/frontend/charter.md`" + ` | Active | |
| Backend | Backend | ` + "`.squad/agents/backend/charter.md`" + ` | Active | |
| Tester | Tester | ` + "`.squad/agents/tester/charter.md`" + ` | Active | |
`
	if err := os.WriteFile(team, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := "CUSTOM COORDINATOR BODY LINE"
	squadPath := filepath.Join(OpencodeAgentsDir(root), "squad.md")
	raw, err := os.ReadFile(squadPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(squadPath, append(raw, []byte("\n"+marker+"\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	lead, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "lead.md"))
	if !strings.Contains(string(lead), "model: anthropic/claude-sonnet-4-5") {
		t.Fatalf("lead override:\n%s", lead)
	}
	front, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "frontend.md"))
	if !strings.Contains(string(front), "model: xai/grok-3") {
		t.Fatalf("frontend inherit:\n%s", front)
	}
	squad, _ := os.ReadFile(squadPath)
	if !strings.Contains(string(squad), "model: xai/grok-3") {
		t.Fatalf("squad splice:\n%s", squad)
	}
	if !strings.Contains(string(squad), marker) {
		t.Fatalf("squad body lost:\n%s", squad)
	}
}

func TestRecastOmitsModelWhenBothEmpty(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	lead, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "lead.md"))
	if strings.Contains(string(lead), "model:") {
		t.Fatalf("unexpected model:\n%s", lead)
	}
	squad, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "squad.md"))
	if strings.Contains(string(squad), "\nmodel:") || strings.HasPrefix(string(squad), "model:") {
		t.Fatalf("squad should have no model:\n%s", squad)
	}
}

func TestUpgradeThenRecastRestoresSquadModel(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	team := filepath.Join(ResolveDir(root), "team.md")
	body := `# Mission Control

## Coordinator

| Name | Role | Model | Notes |
|------|------|-------|-------|
| Squad | Coordinator | xai/grok-3 | Routes |

## Members

| Name | Role | Charter | Status | Model |
|------|------|---------|--------|-------|
| Lead | Lead | ` + "`.squad/agents/lead/charter.md`" + ` | Active | |
| Frontend | Frontend | ` + "`.squad/agents/frontend/charter.md`" + ` | Active | |
| Backend | Backend | ` + "`.squad/agents/backend/charter.md`" + ` | Active | |
| Tester | Tester | ` + "`.squad/agents/tester/charter.md`" + ` | Active | |
`
	if err := os.WriteFile(team, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	if _, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "squad.md"))
	if !strings.Contains(string(got), "model: xai/grok-3") {
		t.Fatalf("upgrade wiped orchestrator model:\n%s", got)
	}
}

func TestSetMemberModelPromotesColumn(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := SetMemberModel(root, "Lead", "anthropic/claude-sonnet-4-5"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(ResolveDir(root), "team.md"))
	if !strings.Contains(string(raw), "Model") || !strings.Contains(string(raw), "anthropic/claude-sonnet-4-5") {
		t.Fatalf("%s", raw)
	}
	members, err := ReadTeam(root)
	if err != nil {
		t.Fatal(err)
	}
	var lead TeamMember
	for _, m := range members {
		if m.ID == "lead" {
			lead = m
		}
	}
	if lead.Model != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("%+v", members)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "lead.md"))
	if !strings.Contains(string(got), "model: anthropic/claude-sonnet-4-5") {
		t.Fatal(string(got))
	}
}

func TestSetSquadModelAndClear(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := SetSquadModel(root, "xai/grok-3"); err != nil {
		t.Fatal(err)
	}
	if ReadSquadModelMust(t, root) != "xai/grok-3" {
		t.Fatal("set")
	}
	if err := SetSquadModel(root, ""); err != nil {
		t.Fatal(err)
	}
	if ReadSquadModelMust(t, root) != "" {
		t.Fatal("clear")
	}
}

func TestAddMemberWithModel(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := AddMember(root, "Designer", "Design", "opencode/gpt-5.1-codex"); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "designer.md"))
	if !strings.Contains(string(got), "model: opencode/gpt-5.1-codex") {
		t.Fatal(string(got))
	}
}

func TestApplyThemeKeepsModelCell(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if err := SetMemberModel(root, "Lead", "anthropic/claude-sonnet-4-5"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyTheme(root, "office"); err != nil {
		t.Fatal(err)
	}
	members, _ := ReadTeam(root)
	found := false
	for _, m := range members {
		if m.ID == "lead" && m.Model == "anthropic/claude-sonnet-4-5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("theme wiped model: %+v", members)
	}
}

func ReadSquadModelMust(t *testing.T, root string) string {
	t.Helper()
	s, err := ReadSquadModel(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRecastThemedHostStillInjects(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Theme: "office"}); err != nil {
		t.Fatal(err)
	}
	// birth office: memory+host id is michael. Set lead/michael model via team.md Model column after parse.
	if err := SetMemberModel(root, "Michael", "anthropic/claude-sonnet-4-5"); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(OpencodeAgentsDir(root), "michael.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "model: anthropic/claude-sonnet-4-5") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(string(got), "You are **Lead**") {
		t.Fatalf("stock prompt lost:\n%s", got)
	}
}
