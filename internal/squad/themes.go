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

	ThemeOriginInit    = "init"
	ThemeOriginApplied = "applied"
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

// HostAgentID returns the OpenCode agent filename stem for a memory id.
// Remaps only when origin is applied and id is in the office theme map.
func HostAgentID(id, theme, origin string) string {
	if origin == ThemeOriginApplied && theme == ThemeOffice {
		if name, ok := officeTheme[id]; ok {
			return strings.ToLower(name)
		}
	}
	return id
}

// OfficeMentionSlug is the extra @tag stem for a stable role id (lead → michael).
// Independent of origin. Empty when the id is not in the office map.
func OfficeMentionSlug(id string) string {
	name, ok := officeTheme[id]
	if !ok {
		return ""
	}
	return strings.ToLower(name)
}

// MentionsPath is .squad/mentions.md under the live squad directory.
func MentionsPath(projectRoot string) string {
	return filepath.Join(ResolveDir(projectRoot), "mentions.md")
}

// WriteMentionMap overwrites mentions.md with the role ↔ tag mapping table.
func WriteMentionMap(projectRoot string, rows []MentionRow) error {
	var b strings.Builder
	b.WriteString("# Mentions\n\n")
	b.WriteString("Theme applied after init. Use **Tag now**. Treat **Was** as the same agent.\n\n")
	b.WriteString("| Role | Tag now | Was |\n")
	b.WriteString("|------|---------|-----|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | @%s | @%s |\n", r.Role, r.Now, r.Was)
	}
	path := MentionsPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// ReadMentionMap parses mentions.md into slug rows (without @).
func ReadMentionMap(projectRoot string) ([]MentionRow, error) {
	raw, err := os.ReadFile(MentionsPath(projectRoot))
	if err != nil {
		return nil, err
	}
	var rows []MentionRow
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := splitTableCells(trimmed)
		if len(cells) < 3 {
			continue
		}
		if strings.EqualFold(cells[0], "Role") || strings.HasPrefix(cells[0], "-") {
			continue
		}
		now := strings.TrimPrefix(cells[1], "@")
		was := strings.TrimPrefix(cells[2], "@")
		rows = append(rows, MentionRow{Role: cells[0], Now: now, Was: was})
	}
	return rows, nil
}

// ClearMentionMap removes mentions.md. Missing file is success.
func ClearMentionMap(projectRoot string) error {
	err := os.Remove(MentionsPath(projectRoot))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func themeNames(theme string) map[string]string {
	if theme == ThemeOffice {
		return officeTheme
	}
	return roleNameByID
}

// ApplyTheme rewrites member display names in team.md and charter titles, then
// records theme and origin on .squad/config.json. Memory IDs are unchanged.
// Applied office writes a mention map; none on applied clears origin and the map.
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
		wasApplied := cfg.ThemeOrigin == ThemeOriginApplied
		cfg.Theme = ""
		if wasApplied {
			cfg.ThemeOrigin = ""
		}
		if err := SaveConfig(projectRoot, *cfg); err != nil {
			return err
		}
		if wasApplied {
			return ClearMentionMap(projectRoot)
		}
		return nil
	}
	cfg.Theme = theme
	if cfg.ThemeOrigin != ThemeOriginInit {
		cfg.ThemeOrigin = ThemeOriginApplied
	}
	if err := SaveConfig(projectRoot, *cfg); err != nil {
		return err
	}
	if cfg.ThemeOrigin == ThemeOriginApplied {
		return WriteMentionMap(projectRoot, appliedOfficeMentionRows())
	}
	return nil
}

func appliedOfficeMentionRows() []MentionRow {
	ids := []string{"lead", "frontend", "backend", "tester"}
	rows := make([]MentionRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, MentionRow{
			Role: roleNameByID[id],
			Now:  OfficeMentionSlug(id),
			Was:  id,
		})
	}
	return rows
}
