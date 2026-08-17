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
	if err := AddMember(root, "Designer", "Design"); err != nil {
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
	if err := AddMember(root, "Lead", "Lead"); err == nil {
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
