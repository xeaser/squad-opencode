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
	if err := AddMember(root, "Designer", "Design"); err != nil {
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
	cfg := Detect(root).Config
	if cfg.Theme != ThemeOffice {
		t.Fatalf("config theme: %+v", cfg)
	}
	if cfg.ThemeOrigin != ThemeOriginApplied {
		t.Fatalf("theme origin: %+v", cfg)
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
	want := []string{"lead", "frontend", "backend", "tester", "designer"}
	if !sameIDs(res.IDs, want) {
		t.Fatalf("recast ids %v want %v", res.IDs, want)
	}
	agents := filepath.Join(root, ".opencode", "agents")
	if _, err := os.Stat(filepath.Join(agents, "michael.md")); err != nil {
		t.Fatal("michael.md must exist after applied office")
	}
	if _, err := os.Stat(filepath.Join(agents, "lead.md")); !os.IsNotExist(err) {
		t.Fatal("lead.md must not exist after applied office")
	}
	if _, err := os.Stat(filepath.Join(agents, "designer.md")); err != nil {
		t.Fatal("designer.md must remain")
	}
	rows, err := ReadMentionMap(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("mention rows: %+v", rows)
	}
	wantNow := map[string]string{"lead": "michael", "frontend": "jim", "backend": "dwight", "tester": "pam"}
	gotNow := map[string]string{}
	for _, r := range rows {
		gotNow[r.Was] = r.Now
	}
	for was, now := range wantNow {
		if gotNow[was] != now {
			t.Fatalf("mention map %s: got %q want %q in %+v", was, gotNow[was], now, rows)
		}
	}

	if err := ApplyTheme(root, "none"); err != nil {
		t.Fatal(err)
	}
	cfg = Detect(root).Config
	if cfg.Theme != "" {
		t.Fatalf("theme none should clear config: %+v", cfg)
	}
	if cfg.ThemeOrigin != "" {
		t.Fatalf("theme origin should clear: %+v", cfg)
	}
	if _, err := os.Stat(MentionsPath(root)); !os.IsNotExist(err) {
		t.Fatal("mentions.md should be gone")
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
	if !sameIDs(res2.IDs, want) {
		t.Fatalf("recast ids after none %v want %v", res2.IDs, want)
	}
	if _, err := os.Stat(filepath.Join(agents, "lead.md")); err != nil {
		t.Fatal("lead.md must exist after none")
	}
	if _, err := os.Stat(filepath.Join(agents, "michael.md")); !os.IsNotExist(err) {
		t.Fatal("michael.md must be gone after none")
	}
	if _, err := os.Stat(filepath.Join(agents, "designer.md")); err != nil {
		t.Fatal("designer.md must remain after none")
	}
}

func sameIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestApplyThemeOfficePreservesInitOrigin(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	cfg := Detect(root).Config
	cfg.Theme = ThemeOffice
	cfg.ThemeOrigin = ThemeOriginInit
	if err := SaveConfig(root, *cfg); err != nil {
		t.Fatal(err)
	}
	if err := ApplyTheme(root, "office"); err != nil {
		t.Fatal(err)
	}
	got := Detect(root).Config
	if got.ThemeOrigin != ThemeOriginInit {
		t.Fatalf("must not overwrite init origin: %+v", got)
	}
	if _, err := os.Stat(MentionsPath(root)); !os.IsNotExist(err) {
		t.Fatal("init origin must not write mention map")
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

func TestNormalizeTheme(t *testing.T) {
	got, err := NormalizeTheme("OFFICE")
	if err != nil || got != ThemeOffice {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeTheme("scranton"); !errors.Is(err, ErrUnknownTheme) {
		t.Fatalf("got %v", err)
	}
}

func TestHostAgentID(t *testing.T) {
	if HostAgentID("lead", ThemeOffice, ThemeOriginApplied) != "michael" {
		t.Fatal()
	}
	if HostAgentID("lead", ThemeOffice, ThemeOriginInit) != "lead" {
		t.Fatal("init origin: id already themed; HostAgentID is the memory id")
	}
	if HostAgentID("lead", "", "") != "lead" {
		t.Fatal()
	}
	if HostAgentID("designer", ThemeOffice, ThemeOriginApplied) != "designer" {
		t.Fatal("unmapped ids unchanged")
	}
}

func TestMentionMapRoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	rows := []MentionRow{{Role: "Lead", Now: "michael", Was: "lead"}}
	if err := WriteMentionMap(root, rows); err != nil {
		t.Fatal(err)
	}
	got, err := ReadMentionMap(root)
	if err != nil || len(got) != 1 || got[0].Now != "michael" || got[0].Was != "lead" {
		t.Fatalf("%v %v", got, err)
	}
	if err := ClearMentionMap(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(MentionsPath(root)); !os.IsNotExist(err) {
		t.Fatal("clear should remove mentions.md")
	}
}
