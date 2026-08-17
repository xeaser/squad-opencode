package squad

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyThemeToTeamMarkdownInvertible(t *testing.T) {
	raw, err := os.ReadFile("templates/squad/team.md")
	if err != nil {
		t.Fatal(err)
	}
	original := string(raw)
	office := applyThemeToTeamMarkdown(original, officeTheme)
	for _, name := range []string{"Michael", "Jim", "Dwight", "Pam"} {
		if !strings.Contains(office, "| "+name+" |") {
			t.Fatalf("office missing %s:\n%s", name, office)
		}
	}
	if strings.Contains(office, "| Lead | Lead |") {
		t.Fatal("office still has role-named Lead row")
	}
	if !strings.Contains(office, "| Squad | Coordinator |") {
		t.Fatal("coordinator must stay Squad")
	}
	restored := applyThemeToTeamMarkdown(office, roleNameByID)
	if restored != original {
		t.Fatalf("apply office then none should restore team.md\n--- got ---\n%s\n--- want ---\n%s", restored, original)
	}
}

func TestParseTeamMarkdownKeepsIDFromCharter(t *testing.T) {
	md := "## Members\n\n| Name | Role | Charter | Status |\n|------|------|---------|--------|\n| Michael | Lead | `.squad/agents/lead/charter.md` | Active |\n"
	members := ParseTeamMarkdown(md)
	if len(members) != 1 {
		t.Fatalf("got %+v", members)
	}
	if members[0].ID != "lead" || members[0].Name != "Michael" || members[0].Role != "Lead" {
		t.Fatalf("id must stay lead: %+v", members[0])
	}
}

func TestApplyThemeOfficeThenNone(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	teamPath := filepath.Join(root, ".squad", "team.md")
	beforeTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeLead, err := os.ReadFile(filepath.Join(root, ".squad", "agents", "lead", "charter.md"))
	if err != nil {
		t.Fatal(err)
	}

	if err := ApplyTheme(root, "office"); err != nil {
		t.Fatal(err)
	}
	if Detect(root).Config.Theme != ThemeOffice {
		t.Fatalf("config theme: %+v", Detect(root).Config)
	}
	officeTeam, _ := os.ReadFile(teamPath)
	if !strings.Contains(string(officeTeam), "| Michael | Lead |") {
		t.Fatalf("team.md:\n%s", officeTeam)
	}
	leadCharter, _ := os.ReadFile(filepath.Join(root, ".squad", "agents", "lead", "charter.md"))
	if !strings.HasPrefix(string(leadCharter), "# Michael\n") {
		t.Fatalf("charter title:\n%s", leadCharter)
	}

	res, err := Recast(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"lead", "frontend", "backend", "tester"}
	for _, id := range want {
		if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", id+".md")); err != nil {
			t.Fatalf("missing host agent %s: %v", id, err)
		}
	}
	for id, name := range officeTheme {
		slug := memberID(name)
		alias := filepath.Join(root, ".opencode", "agents", slug+".md")
		if _, err := os.Stat(alias); err != nil {
			t.Fatalf("missing @mention file %s: %v", slug, err)
		}
		idBody, _ := os.ReadFile(filepath.Join(root, ".opencode", "agents", id+".md"))
		aliasBody, _ := os.ReadFile(alias)
		if string(idBody) != string(aliasBody) {
			t.Fatalf("%s.md and %s.md should match", id, slug)
		}
	}
	if !containsAll(res.IDs, append(append([]string{}, want...), "michael", "jim", "dwight", "pam")...) {
		t.Fatalf("recast ids %v", res.IDs)
	}

	if err := ApplyTheme(root, "none"); err != nil {
		t.Fatal(err)
	}
	if Detect(root).Config.Theme != "" {
		t.Fatalf("theme none should clear config: %+v", Detect(root).Config)
	}
	afterTeam, _ := os.ReadFile(teamPath)
	if string(afterTeam) != string(beforeTeam) {
		t.Fatalf("team.md not restored\n--- got ---\n%s\n--- want ---\n%s", afterTeam, beforeTeam)
	}
	afterLead, _ := os.ReadFile(filepath.Join(root, ".squad", "agents", "lead", "charter.md"))
	if string(afterLead) != string(beforeLead) {
		t.Fatalf("charter not restored\n--- got ---\n%s\n--- want ---\n%s", afterLead, beforeLead)
	}
	res2, err := Recast(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(res2.IDs, want...) {
		t.Fatalf("recast ids after none %v want %v", res2.IDs, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "michael.md")); !os.IsNotExist(err) {
		t.Fatal("theme none should remove michael.md")
	}
}

func containsAll(got []string, want ...string) bool {
	have := map[string]bool{}
	for _, s := range got {
		have[s] = true
	}
	for _, s := range want {
		if !have[s] {
			return false
		}
	}
	return true
}

func TestApplyThemeUnknown(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	err := ApplyTheme(root, "parks")
	if !errors.Is(err, ErrUnknownTheme) {
		t.Fatalf("got %v", err)
	}
}

func TestOfficeMentionSlug(t *testing.T) {
	if got := OfficeMentionSlug("lead"); got != "michael" {
		t.Fatalf("got %q", got)
	}
	if got := OfficeMentionSlug("designer"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeTheme(t *testing.T) {
	got, err := NormalizeTheme("OFFICE")
	if err != nil || got != ThemeOffice {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeTheme("scranton"); !errors.Is(err, ErrUnknownTheme) {
		t.Fatalf("got %v", err)
	}
}
