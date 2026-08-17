package squad

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// (?m) so ^ matches line starts (Go default is whole-string only).
	reMembersHeading = regexp.MustCompile(`(?mi)^##\s+Members\b`)
	reAnyHeading     = regexp.MustCompile(`(?m)^##\s+`)
	reTableSep       = regexp.MustCompile(`(?m)^\|\s*-+`)
	reNameHeader     = regexp.MustCompile(`(?mi)^\|\s*Name\s*\|`)
)

// ParseTeamMarkdown parses member rows from team.md.
// Prefer the "## Members" section when present.
func ParseTeamMarkdown(content string) []TeamMember {
	hasMembers := reMembersHeading.MatchString(content)
	inMembers := !hasMembers
	var members []TeamMember

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSuffix(trimmed, "\r")

		if reMembersHeading.MatchString(trimmed) {
			inMembers = true
			continue
		}
		if hasMembers && reAnyHeading.MatchString(trimmed) && !reMembersHeading.MatchString(trimmed) {
			inMembers = false
			continue
		}
		if !inMembers {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if reTableSep.MatchString(trimmed) || reNameHeader.MatchString(trimmed) {
			continue
		}

		cells := splitTableCells(trimmed)
		if len(cells) < 2 {
			continue
		}
		name := cells[0]
		role := cells[1]
		if strings.EqualFold(name, "name") {
			continue
		}
		status := "Active"
		if len(cells) >= 4 {
			status = cells[3]
		} else if len(cells) >= 3 {
			status = cells[len(cells)-1]
		}
		id := memberIDFromRow(name, cells)
		members = append(members, TeamMember{
			ID:     id,
			Name:   name,
			Role:   role,
			Status: status,
		})
	}
	return members
}

// memberIDFromRow prefers the charter path (`.squad/agents/<id>/…`) so display
// names can change without renaming @lead / recast files.
func memberIDFromRow(name string, cells []string) string {
	if len(cells) >= 3 {
		if id := idFromCharter(cells[2]); id != "" {
			return id
		}
	}
	return memberID(name)
}

func idFromCharter(cell string) string {
	s := strings.Trim(cell, "`")
	s = strings.ReplaceAll(s, "\\", "/")
	const marker = "agents/"
	i := strings.Index(s, marker)
	if i < 0 {
		return ""
	}
	rest := s[i+len(marker):]
	id, _, _ := strings.Cut(rest, "/")
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" || id == "squad" {
		return ""
	}
	return id
}

func splitTableCells(line string) []string {
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cells = append(cells, p)
		}
	}
	return cells
}

// ReadTeam loads and parses .squad/team.md.
func ReadTeam(projectRoot string) ([]TeamMember, error) {
	path := filepath.Join(ResolveDir(projectRoot), "team.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseTeamMarkdown(string(data)), nil
}
