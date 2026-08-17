package squad

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Theme ids. v1 ships office only; "none" restores role names by member id.
const (
	ThemeOffice = "office"
	ThemeNone   = "none"
)

// ErrUnknownTheme is returned for --theme values other than office|none.
var ErrUnknownTheme = errors.New("unknown theme")

// officeTheme maps stable member IDs to Office display names.
// Coordinator (squad) is not a character.
var officeTheme = map[string]string{
	"lead":     "Michael",
	"frontend": "Jim",
	"backend":  "Dwight",
	"tester":   "Pam",
}

// roleNameByID is the default display name restored by --theme none.
var roleNameByID = map[string]string{
	"lead":     "Lead",
	"frontend": "Frontend",
	"backend":  "Backend",
	"tester":   "Tester",
}

// NormalizeTheme accepts office|none (any case). Unknown values are ErrUnknownTheme.
func NormalizeTheme(theme string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(theme))
	switch t {
	case ThemeOffice, ThemeNone:
		return t, nil
	default:
		return "", fmt.Errorf("%w: %s (want office or none)", ErrUnknownTheme, theme)
	}
}

func themeNames(theme string) map[string]string {
	if theme == ThemeOffice {
		return officeTheme
	}
	return roleNameByID
}

// OfficeMentionSlug is the extra @tag file stem for a stable role id
// (lead → michael). Empty when the slug would match the id.
func OfficeMentionSlug(id string) string {
	name, ok := officeTheme[id]
	if !ok {
		return ""
	}
	slug := memberID(name)
	if slug == "" || slug == id {
		return ""
	}
	return slug
}

func officeMentionSlugs() []string {
	var out []string
	for id := range officeTheme {
		if s := OfficeMentionSlug(id); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ApplyTheme rewrites member display names in team.md and charter titles, then
// records theme on .squad/config.json. Agent IDs are unchanged.
func ApplyTheme(projectRoot, theme string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	norm, err := NormalizeTheme(theme)
	if err != nil {
		return err
	}
	names := themeNames(norm)
	teamFile := filepath.Join(ResolveDir(projectRoot), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	next := applyThemeToTeamMarkdown(string(raw), names)
	if err := os.WriteFile(teamFile, []byte(next), 0o644); err != nil {
		return err
	}
	if err := applyThemeToCharters(projectRoot, names); err != nil {
		return err
	}
	return setConfigTheme(projectRoot, norm)
}

// applyThemeToTeamMarkdown rewrites the Name cell of member rows whose id is in names.
func applyThemeToTeamMarkdown(content string, names map[string]string) string {
	hasMembers := reMembersHeading.MatchString(content)
	inMembers := !hasMembers
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if reMembersHeading.MatchString(trimmed) {
			inMembers = true
			continue
		}
		if hasMembers && reAnyHeading.MatchString(trimmed) && !reMembersHeading.MatchString(trimmed) {
			inMembers = false
			continue
		}
		if !inMembers || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if reTableSep.MatchString(trimmed) || reNameHeader.MatchString(trimmed) {
			continue
		}
		cells := splitTableCells(trimmed)
		if len(cells) < 2 {
			continue
		}
		oldName := cells[0]
		if strings.EqualFold(oldName, "name") {
			continue
		}
		id := memberIDFromRow(oldName, cells)
		newName, ok := names[id]
		if !ok || newName == oldName {
			continue
		}
		lines[i] = replaceNameCell(line, oldName, newName)
	}
	return strings.Join(lines, "\n")
}

func replaceNameCell(line, oldName, newName string) string {
	idx := strings.Index(line, "|")
	if idx < 0 {
		return line
	}
	rest := line[idx+1:]
	cellEnd := strings.Index(rest, "|")
	if cellEnd < 0 {
		return line
	}
	cell := rest[:cellEnd]
	newCell := strings.Replace(cell, oldName, newName, 1)
	return line[:idx+1] + newCell + rest[cellEnd:]
}

func applyThemeToCharters(projectRoot string, names map[string]string) error {
	base := filepath.Join(ResolveDir(projectRoot), "agents")
	for id, title := range names {
		path := filepath.Join(base, id, "charter.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		next := applyThemeToCharterTitle(string(raw), title)
		if next == string(raw) {
			continue
		}
		if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func applyThemeToCharterTitle(content, title string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(trim, "# ") && !strings.HasPrefix(trim, "##") {
			prefix := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = prefix + "# " + title
			break
		}
	}
	return strings.Join(lines, "\n")
}

func setConfigTheme(projectRoot, theme string) error {
	cfg := Detect(projectRoot).Config
	if cfg == nil {
		return fmt.Errorf("not initialized")
	}
	if theme == ThemeNone {
		cfg.Theme = ""
	} else {
		cfg.Theme = theme
	}
	return SaveConfig(projectRoot, *cfg)
}
