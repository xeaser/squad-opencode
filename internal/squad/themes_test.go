package squad

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyThemeToCharterTitlePreservesCRLF(t *testing.T) {
	original := "# Lead\r\n\r\n## Mission\r\n"
	office := applyThemeToCharterTitle(original, "Michael")
	if office != "# Michael\r\n\r\n## Mission\r\n" {
		t.Fatalf("office:\n%q", office)
	}
	restored := applyThemeToCharterTitle(office, "Lead")
	if restored != original {
		t.Fatalf("restored:\n%q want %q", restored, original)
	}
}

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
	if err := AddMember(root, "Designer", "Design", ""); err != nil {
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
	if !strings.Contains(string(officeTeam), "@michael") || strings.Contains(string(officeTeam), "@lead") {
		t.Fatalf("How to work should list @michael not @lead:\n%s", officeTeam)
	}
	leadCharter, _ := os.ReadFile(filepath.Join(root, ".squad", "agents", "lead", "charter.md"))
	if !strings.HasPrefix(strings.ReplaceAll(string(leadCharter), "\r\n", "\n"), "# Michael\n") {
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
	if !strings.Contains(string(afterTeam), "@lead") || strings.Contains(string(afterTeam), "@michael") {
		t.Fatalf("How to work should restore @lead:\n%s", afterTeam)
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

func TestApplyThemeNoneAfterInitOrigin(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, Theme: ThemeOffice}); err != nil {
		t.Fatal(err)
	}
	if err := AddMember(root, "Designer", "Design", ""); err != nil {
		t.Fatal(err)
	}

	if err := ApplyTheme(root, "none"); err != nil {
		t.Fatal(err)
	}
	cfg := Detect(root).Config
	if cfg.Theme != "" || cfg.ThemeOrigin != "" {
		t.Fatalf("none after init must clear theme and origin: %+v", cfg)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "lead", "charter.md")); err != nil {
		t.Fatal("must rename agents/michael back to lead")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "agents", "michael")); !os.IsNotExist(err) {
		t.Fatal("birth none must not keep agents/michael")
	}
	if _, err := os.Stat(MentionsPath(root)); !os.IsNotExist(err) {
		t.Fatal("mentions.md must stay gone")
	}
	team, err := os.ReadFile(filepath.Join(root, ".squad", "team.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(team)
	if !strings.Contains(body, "| Lead | Lead |") || strings.Contains(body, "| Michael | Lead |") {
		t.Fatalf("names:\n%s", body)
	}
	if !strings.Contains(body, ".squad/agents/lead/") || strings.Contains(body, ".squad/agents/michael/") {
		t.Fatalf("charter paths:\n%s", body)
	}
	if !strings.Contains(body, "@lead") || strings.Contains(body, "@michael") {
		t.Fatalf("How to work:\n%s", body)
	}
	lead, err := os.ReadFile(filepath.Join(root, ".squad", "agents", "lead", "charter.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.ReplaceAll(string(lead), "\r\n", "\n"), "# Lead\n") {
		t.Fatalf("charter title:\n%s", lead)
	}
	if strings.Contains(string(lead), ".squad/agents/michael/") {
		t.Fatalf("charter still points at michael:\n%s", lead)
	}

	res, err := Recast(root)
	if err != nil {
		t.Fatal(err)
	}
	if !sameIDs(res.IDs, []string{"lead", "frontend", "backend", "tester", "designer"}) {
		t.Fatalf("recast ids %v", res.IDs)
	}
	agents := filepath.Join(root, ".opencode", "agents")
	if _, err := os.Stat(filepath.Join(agents, "lead.md")); err != nil {
		t.Fatal("lead.md must exist after none")
	}
	if _, err := os.Stat(filepath.Join(agents, "michael.md")); !os.IsNotExist(err) {
		t.Fatal("michael.md must be gone")
	}
	if _, err := os.Stat(filepath.Join(agents, "designer.md")); err != nil {
		t.Fatal("designer.md must remain")
	}
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

func TestInitAndUpgradeTemplatesMentionMap(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}

	assertMentionCopy := func(when string) {
		t.Helper()
		for _, rel := range []string{
			".opencode/agents/squad.md",
			".opencode/skills/squad-team/SKILL.md",
		} {
			got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("%s read %s: %v", when, rel, err)
			}
			body := string(got)
			for _, want := range []string{"mentions.md", "Tag now", "team.md"} {
				if !strings.Contains(body, want) {
					t.Errorf("%s %s missing %q", when, rel, want)
				}
			}
			if strings.Contains(body, "Otherwise `@lead`") || strings.Contains(body, "`@frontend` / `@backend` / `@tester` / `@lead`") {
				t.Errorf("%s %s hardcodes @lead fallback", when, rel)
			}
		}
	}

	assertMentionCopy("init")

	if _, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	assertMentionCopy("upgrade")
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
