package squad

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// (?m) so ^ matches line starts (Go default is whole-string only).
	reMembersHeading     = regexp.MustCompile(`(?mi)^##\s+Members\b`)
	reCoordinatorHeading = regexp.MustCompile(`(?mi)^##\s+Coordinator\b`)
	reAnyHeading         = regexp.MustCompile(`(?m)^##\s+`)
	reTableSep           = regexp.MustCompile(`(?m)^\|\s*-+`)
	reNameHeader         = regexp.MustCompile(`(?mi)^\|\s*Name\s*\|`)
)

type tableRow struct {
	cells []string
	cols  map[string]int
}

// ParseTeamMarkdown parses member rows from team.md.
// Prefer the "## Members" section when present.
func ParseTeamMarkdown(content string) []TeamMember {
	var section *regexp.Regexp
	if reMembersHeading.MatchString(content) {
		section = reMembersHeading
	}
	var members []TeamMember
	for _, row := range walkMarkdownTableRows(content, section) {
		if len(row.cells) < 2 {
			continue
		}
		name := cellNamed(row, "name", 0)
		if strings.EqualFold(name, "name") {
			continue
		}
		role := cellNamed(row, "role", 1)
		status := "Active"
		if s, ok := namedCell(row, "status"); ok {
			if s != "" {
				status = s
			}
		} else if row.cols == nil {
			if len(row.cells) >= 4 {
				status = row.cells[3]
			} else if len(row.cells) >= 3 {
				status = row.cells[len(row.cells)-1]
			}
		}
		model, _ := namedCell(row, "model")
		members = append(members, TeamMember{
			ID:     rowMemberID(name, row),
			Name:   name,
			Role:   role,
			Status: status,
			Model:  model,
		})
	}
	return members
}

// ParseSquadModel returns the Coordinator Squad row's Model cell ("" if absent).
func ParseSquadModel(content string) string {
	if !reCoordinatorHeading.MatchString(content) {
		return ""
	}
	var rows []tableRow
	for _, row := range walkMarkdownTableRows(content, reCoordinatorHeading) {
		if len(row.cells) == 0 {
			continue
		}
		if strings.EqualFold(cellNamed(row, "name", 0), "name") {
			continue
		}
		rows = append(rows, row)
	}
	var chosen *tableRow
	for i := range rows {
		if strings.EqualFold(cellNamed(rows[i], "name", 0), "squad") {
			chosen = &rows[i]
			break
		}
	}
	if chosen == nil && len(rows) == 1 {
		chosen = &rows[0]
	}
	if chosen == nil {
		return ""
	}
	model, _ := namedCell(*chosen, "model")
	return model
}

// ReadSquadModel loads the Coordinator model from .squad/team.md.
// A missing file is "", nil.
func ReadSquadModel(projectRoot string) (string, error) {
	path := filepath.Join(ResolveDir(projectRoot), "team.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return ParseSquadModel(string(data)), nil
}

// EffectiveModel prefers a member's own model, else the team/squad model.
func EffectiveModel(own, team string) string {
	if s := strings.TrimSpace(own); s != "" {
		return s
	}
	return strings.TrimSpace(team)
}

func walkMarkdownTableRows(content string, section *regexp.Regexp) []tableRow {
	restrict := section != nil
	inSection := !restrict
	var cols map[string]int
	var rows []tableRow
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if section != nil && section.MatchString(trimmed) {
			inSection = true
			cols = nil
			continue
		}
		if restrict && reAnyHeading.MatchString(trimmed) && !section.MatchString(trimmed) {
			inSection = false
			cols = nil
			continue
		}
		if !inSection || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if reTableSep.MatchString(trimmed) {
			continue
		}
		cells := splitTableCells(trimmed)
		if len(cells) == 0 {
			continue
		}
		if strings.EqualFold(cells[0], "name") || reNameHeader.MatchString(trimmed) {
			cols = headerColumns(cells)
			continue
		}
		rows = append(rows, tableRow{cells: cells, cols: cols})
	}
	return rows
}

func headerColumns(cells []string) map[string]int {
	cols := make(map[string]int, len(cells))
	for i, c := range cells {
		cols[strings.ToLower(c)] = i
	}
	return cols
}

func namedCell(row tableRow, name string) (string, bool) {
	if row.cols == nil {
		return "", false
	}
	i, ok := row.cols[name]
	if !ok {
		return "", false
	}
	return cellAt(row.cells, i), true
}

func cellNamed(row tableRow, name string, fallback int) string {
	if s, ok := namedCell(row, name); ok {
		return s
	}
	return cellAt(row.cells, fallback)
}

func cellAt(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}

func rowMemberID(name string, row tableRow) string {
	charter := cellNamed(row, "charter", 2)
	if id := idFromCharter(charter); id != "" {
		return id
	}
	return memberID(name)
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
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
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
