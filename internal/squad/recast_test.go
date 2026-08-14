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
