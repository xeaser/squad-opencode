package squad

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RecastResult is how many host agent files were written.
type RecastResult struct {
	Written int
	IDs     []string
}

// AddMember appends a row to team.md and creates charter/knowledge if missing.
// model is optional; empty keeps today's 4-column Members append.
func AddMember(projectRoot, name, role, model string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
	model = strings.TrimSpace(model)
	if model != "" {
		stored, err := ValidateModelID(model)
		if err != nil {
			return err
		}
		model = stored
	}
	if name == "" {
		return fmt.Errorf("name required")
	}
	if role == "" {
		role = name
	}
	id := memberID(name)
	if id == "squad" {
		return fmt.Errorf("%q is reserved for the coordinator agent", name)
	}
	members, err := ReadTeam(projectRoot)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.ID == id {
			return fmt.Errorf("member %q already exists", name)
		}
	}
	teamFile := filepath.Join(ResolveDir(projectRoot), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	next, err := appendMemberRow(string(raw), name, role, id, model)
	if err != nil {
		return err
	}
	if err := os.WriteFile(teamFile, []byte(next), 0o644); err != nil {
		return err
	}
	dir := filepath.Join(ResolveDir(projectRoot), "agents", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	charter := filepath.Join(dir, "charter.md")
	if _, err := os.Stat(charter); os.IsNotExist(err) {
		if err := os.WriteFile(charter, []byte(defaultCharter(name, role, id)), 0o644); err != nil {
			return err
		}
	}
	know := filepath.Join(dir, "knowledge.md")
	if _, err := os.Stat(know); os.IsNotExist(err) {
		body := "# " + name + " knowledge\n\nWhat I've learned about this project (append over time).\n"
		if err := os.WriteFile(know, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// RemoveMember drops a team.md row and the host agent file. Knowledge stays.
func RemoveMember(projectRoot, name string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	id := memberID(name)
	if id == "squad" {
		return fmt.Errorf("%q is reserved for the coordinator agent", name)
	}
	members, err := ReadTeam(projectRoot)
	if err != nil {
		return err
	}
	var found TeamMember
	ok := false
	for _, m := range members {
		if m.ID == id || strings.EqualFold(m.Name, name) {
			found = m
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("member %q not found", name)
	}
	teamFile := filepath.Join(ResolveDir(projectRoot), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	next, err := removeMemberRow(string(raw), found)
	if err != nil {
		return err
	}
	if err := os.WriteFile(teamFile, []byte(next), 0o644); err != nil {
		return err
	}
	theme, origin := "", ""
	if det := Detect(projectRoot); det.Config != nil {
		theme = det.Config.Theme
		origin = det.Config.ThemeOrigin
	}
	host := filepath.Join(OpencodeAgentsDir(projectRoot), HostAgentID(found.ID, theme, origin)+".md")
	if err := os.Remove(host); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ValidateModelID trims a model id. Empty or "-" store as inherit/clear.
// Any other value must contain "/" (provider/model-id).
func ValidateModelID(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "-" {
		return "", nil
	}
	if !strings.Contains(s, "/") {
		return "", fmt.Errorf("model %q must be provider/model-id", raw)
	}
	return s, nil
}

// SetMemberModel writes a Members Model cell. name matches like RemoveMember.
// Empty or "-" clears the cell. Missing Model column is promoted first.
func SetMemberModel(projectRoot, name, model string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	id := memberID(name)
	if id == "squad" {
		return fmt.Errorf("%q is reserved for the coordinator agent", name)
	}
	stored, err := ValidateModelID(model)
	if err != nil {
		return err
	}
	members, err := ReadTeam(projectRoot)
	if err != nil {
		return err
	}
	var found TeamMember
	ok := false
	for _, m := range members {
		if m.ID == id || strings.EqualFold(m.Name, name) {
			found = m
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("member %q not found", name)
	}
	teamFile := filepath.Join(ResolveDir(projectRoot), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	next, err := setMemberModelCell(string(raw), found, stored)
	if err != nil {
		return err
	}
	return os.WriteFile(teamFile, []byte(next), 0o644)
}

// SetSquadModel writes the Coordinator Squad Model cell. Empty or "-" clears it.
// Missing Model column is promoted first.
func SetSquadModel(projectRoot, model string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	stored, err := ValidateModelID(model)
	if err != nil {
		return err
	}
	teamFile := filepath.Join(ResolveDir(projectRoot), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	next, err := setSquadModelCell(string(raw), stored)
	if err != nil {
		return err
	}
	return os.WriteFile(teamFile, []byte(next), 0o644)
}

func memberID(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

func appendMemberRow(content, name, role, id, model string) (string, error) {
	if model != "" {
		var err error
		content, err = ensureSectionModelColumn(content, reMembersHeading, false)
		if err != nil {
			return "", err
		}
	}
	hasModel := sectionHeaderHasModel(content, reMembersHeading, false)
	row := fmt.Sprintf("| %s | %s | `.squad/agents/%s/charter.md` | Active |", name, role, id)
	if hasModel {
		row = fmt.Sprintf("| %s | %s | `.squad/agents/%s/charter.md` | Active | %s |", name, role, id, model)
	}
	lines := strings.Split(content, "\n")
	inMembers := false
	lastTable := -1
	for i, line := range lines {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if reMembersHeading.MatchString(trim) {
			inMembers = true
			continue
		}
		if inMembers && reAnyHeading.MatchString(trim) && !reMembersHeading.MatchString(trim) {
			inMembers = false
			if lastTable >= 0 {
				return insertLine(lines, lastTable+1, row), nil
			}
		}
		if inMembers && strings.HasPrefix(trim, "|") {
			lastTable = i
		}
	}
	if lastTable >= 0 {
		return insertLine(lines, lastTable+1, row), nil
	}
	header := "| Name | Role | Charter | Status |"
	sep := "|------|------|---------|--------|"
	if model != "" {
		header = "| Name | Role | Charter | Status | Model |"
		sep = "|------|------|---------|--------|-------|"
		row = fmt.Sprintf("| %s | %s | `.squad/agents/%s/charter.md` | Active | %s |", name, role, id, model)
	}
	block := "\n## Members\n\n" + header + "\n" + sep + "\n" + row + "\n"
	return strings.TrimRight(content, "\n") + block, nil
}

func insertLine(lines []string, at int, row string) string {
	if at > len(lines) {
		at = len(lines)
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, row)
	out = append(out, lines[at:]...)
	return strings.Join(out, "\n")
}

func removeMemberRow(content string, member TeamMember) (string, error) {
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
		name := cells[0]
		rowID := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if rowID == member.ID || strings.EqualFold(name, member.Name) {
			out := make([]string, 0, len(lines)-1)
			out = append(out, lines[:i]...)
			out = append(out, lines[i+1:]...)
			return strings.Join(out, "\n"), nil
		}
	}
	return "", fmt.Errorf("member %q not found", member.Name)
}

func setMemberModelCell(content string, member TeamMember, model string) (string, error) {
	lines := strings.Split(content, "\n")
	headerIdx, sepIdx, dataIdx, cols := locateSectionTable(lines, reMembersHeading, false)
	if headerIdx < 0 {
		return "", fmt.Errorf("members table not found")
	}
	if _, ok := cols["model"]; !ok {
		if model == "" {
			return content, nil
		}
		cols, lines = promoteModelColumn(lines, headerIdx, sepIdx, dataIdx)
	}
	modelIdx, ok := cols["model"]
	if !ok {
		return "", fmt.Errorf("model column missing")
	}
	for _, i := range dataIdx {
		cells := splitTableCells(strings.TrimSpace(strings.TrimSuffix(lines[i], "\r")))
		if rowMatchesMember(cells, member) {
			lines[i] = setTableCell(lines[i], modelIdx, model)
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", fmt.Errorf("member %q not found", member.Name)
}

func setSquadModelCell(content string, model string) (string, error) {
	if !reCoordinatorHeading.MatchString(content) {
		return "", fmt.Errorf("coordinator table not found")
	}
	lines := strings.Split(content, "\n")
	headerIdx, sepIdx, dataIdx, cols := locateSectionTable(lines, reCoordinatorHeading, true)
	if headerIdx < 0 {
		return "", fmt.Errorf("coordinator table not found")
	}
	if _, ok := cols["model"]; !ok {
		if model == "" {
			return content, nil
		}
		cols, lines = promoteModelColumn(lines, headerIdx, sepIdx, dataIdx)
	}
	modelIdx, ok := cols["model"]
	if !ok {
		return "", fmt.Errorf("model column missing")
	}
	chosen := -1
	for _, i := range dataIdx {
		cells := splitTableCells(strings.TrimSpace(strings.TrimSuffix(lines[i], "\r")))
		if len(cells) > 0 && strings.EqualFold(cells[0], "squad") {
			chosen = i
			break
		}
	}
	if chosen < 0 && len(dataIdx) == 1 {
		chosen = dataIdx[0]
	}
	if chosen < 0 {
		return "", fmt.Errorf("coordinator squad row not found")
	}
	lines[chosen] = setTableCell(lines[chosen], modelIdx, model)
	return strings.Join(lines, "\n"), nil
}

func ensureSectionModelColumn(content string, section *regexp.Regexp, requireHeading bool) (string, error) {
	lines := strings.Split(content, "\n")
	headerIdx, sepIdx, dataIdx, cols := locateSectionTable(lines, section, requireHeading)
	if headerIdx < 0 {
		return content, nil
	}
	if _, ok := cols["model"]; ok {
		return content, nil
	}
	_, lines = promoteModelColumn(lines, headerIdx, sepIdx, dataIdx)
	return strings.Join(lines, "\n"), nil
}

func sectionHeaderHasModel(content string, section *regexp.Regexp, requireHeading bool) bool {
	lines := strings.Split(content, "\n")
	_, _, _, cols := locateSectionTable(lines, section, requireHeading)
	if cols == nil {
		return false
	}
	_, ok := cols["model"]
	return ok
}

func locateSectionTable(lines []string, section *regexp.Regexp, requireHeading bool) (headerIdx, sepIdx int, dataIdx []int, cols map[string]int) {
	headerIdx, sepIdx = -1, -1
	hasHeading := false
	if section != nil {
		for _, line := range lines {
			if section.MatchString(strings.TrimSpace(strings.TrimSuffix(line, "\r"))) {
				hasHeading = true
				break
			}
		}
	}
	inSection := section == nil || (!requireHeading && !hasHeading)
	for i, line := range lines {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if section != nil && section.MatchString(trim) {
			inSection = true
			headerIdx, sepIdx = -1, -1
			dataIdx = nil
			cols = nil
			continue
		}
		if hasHeading && reAnyHeading.MatchString(trim) && (section == nil || !section.MatchString(trim)) {
			inSection = false
			continue
		}
		if !inSection || !strings.HasPrefix(trim, "|") {
			continue
		}
		if reTableSep.MatchString(trim) {
			if headerIdx >= 0 && sepIdx < 0 {
				sepIdx = i
			}
			continue
		}
		cells := splitTableCells(trim)
		if len(cells) == 0 {
			continue
		}
		if strings.EqualFold(cells[0], "name") || reNameHeader.MatchString(trim) {
			headerIdx = i
			cols = headerColumns(cells)
			dataIdx = nil
			continue
		}
		if headerIdx >= 0 {
			dataIdx = append(dataIdx, i)
		}
	}
	return
}

func promoteModelColumn(lines []string, headerIdx, sepIdx int, dataIdx []int) (map[string]int, []string) {
	if headerIdx >= 0 {
		lines[headerIdx] = appendTableCell(lines[headerIdx], "Model", false)
	}
	if sepIdx >= 0 {
		lines[sepIdx] = appendTableCell(lines[sepIdx], "-------", true)
	}
	for _, i := range dataIdx {
		lines[i] = appendTableCell(lines[i], "", false)
	}
	var cols map[string]int
	if headerIdx >= 0 {
		trim := strings.TrimSpace(strings.TrimSuffix(lines[headerIdx], "\r"))
		cols = headerColumns(splitTableCells(trim))
	}
	return cols, lines
}

func appendTableCell(line, cell string, sep bool) string {
	eol := ""
	if strings.HasSuffix(line, "\r") {
		eol = "\r"
		line = strings.TrimSuffix(line, "\r")
	}
	s := strings.TrimRight(line, " \t")
	if !strings.HasSuffix(s, "|") {
		s += " |"
	}
	if sep {
		if cell == "" {
			cell = "-------"
		}
		return s + cell + "|" + eol
	}
	if cell == "" {
		return s + "  |" + eol
	}
	return s + " " + cell + " |" + eol
}

func setTableCell(line string, idx int, value string) string {
	eol := ""
	if strings.HasSuffix(line, "\r") {
		eol = "\r"
		line = strings.TrimSuffix(line, "\r")
	}
	cells := splitTableCells(line)
	for len(cells) <= idx {
		cells = append(cells, "")
	}
	cells[idx] = value
	return "| " + strings.Join(cells, " | ") + " |" + eol
}

func rowMatchesMember(cells []string, member TeamMember) bool {
	if len(cells) == 0 {
		return false
	}
	name := cells[0]
	rowID := memberIDFromRow(name, cells)
	return rowID == member.ID || strings.EqualFold(name, member.Name)
}

func defaultCharter(name, role, id string) string {
	return fmt.Sprintf(`# %s

## Mission

Own %s work for this project.

## In scope

- Work assigned to the %s role

## Out of scope

- Merging without human approval

## Knowledge

Write lasting notes to `+"`"+`.squad/agents/%s/knowledge.md`+"`"+`.
`, name, role, role, id)
}

// Recast writes .opencode/agents/<host-id>.md from the live team and splices
// model: onto squad.md (coordinator body is otherwise left alone).
func Recast(projectRoot string) (RecastResult, error) {
	if !IsInitialized(projectRoot) {
		return RecastResult{}, fmt.Errorf("not initialized")
	}
	members, err := ReadTeam(projectRoot)
	if err != nil {
		return RecastResult{}, err
	}
	squadModel, err := ReadSquadModel(projectRoot)
	if err != nil {
		return RecastResult{}, err
	}
	destDir := OpencodeAgentsDir(projectRoot)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return RecastResult{}, err
	}
	theme, origin := "", ""
	if det := Detect(projectRoot); det.Config != nil {
		theme = det.Config.Theme
		origin = det.Config.ThemeOrigin
	}
	liveIDs := map[string]struct{}{}
	hostIDs := map[string]struct{}{}
	var res RecastResult
	tpl := TemplateFS()
	for _, m := range members {
		if m.ID == "" || m.ID == "squad" {
			continue
		}
		body, err := agentMarkdown(tpl, m, squadModel)
		if err != nil {
			return res, err
		}
		host := HostAgentID(m.ID, theme, origin)
		path := filepath.Join(destDir, host+".md")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return res, err
		}
		res.Written++
		res.IDs = append(res.IDs, m.ID)
		liveIDs[m.ID] = struct{}{}
		hostIDs[host] = struct{}{}
	}
	if err := pruneStaleHostAgents(destDir, liveIDs, hostIDs); err != nil {
		return res, err
	}
	if err := spliceSquadModel(projectRoot, squadModel); err != nil {
		return res, err
	}
	return res, nil
}

func pruneStaleHostAgents(destDir string, liveIDs, hostIDs map[string]struct{}) error {
	protected := map[string]struct{}{
		"squad":     {},
		"designer":  {},
		"liveprobe": {},
	}
	keep := map[string]struct{}{}
	for name := range hostIDs {
		keep[name] = struct{}{}
	}
	for name := range liveIDs {
		if _, stock := roleNameByID[name]; stock {
			continue
		}
		keep[name] = struct{}{}
	}
	for name := range protected {
		keep[name] = struct{}{}
	}
	var candidates []string
	for id := range roleNameByID {
		candidates = append(candidates, id)
		if slug := OfficeMentionSlug(id); slug != "" {
			candidates = append(candidates, slug)
		}
	}
	for _, name := range candidates {
		if _, ok := keep[name]; ok {
			continue
		}
		err := os.Remove(filepath.Join(destDir, name+".md"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func stockRoleID(id string) string {
	for roleID, name := range officeTheme {
		if memberID(name) == id {
			return roleID
		}
	}
	return id
}

func agentMarkdown(tpl fs.FS, m TeamMember, squadModel string) (string, error) {
	role := stockRoleID(m.ID)
	var body string
	data, err := fs.ReadFile(tpl, "opencode/agents/"+role+".md")
	if err == nil {
		body = string(data)
		if role != m.ID {
			body = strings.ReplaceAll(body, ".squad/agents/"+role+"/", ".squad/agents/"+m.ID+"/")
		}
	} else {
		body = fmt.Sprintf(`---
description: %s — %s
mode: subagent
permission:
  edit: allow
  bash: allow
---

You are **%s** (%s) on this project's Squad team.

## Always

1. Read `+"`"+`.squad/agents/%s/charter.md`+"`"+` and your `+"`"+`knowledge.md`+"`"+`.
2. Stay within your role unless the human asks otherwise.
3. After meaningful work, append learnings to `+"`"+`.squad/agents/%s/knowledge.md`+"`"+`.

## Escalate to the human

Security-sensitive changes, unclear product priorities, or work outside your charter.
`, m.Name, m.Role, m.Name, m.Role, m.ID, m.ID)
	}
	if effective := EffectiveModel(m.Model, squadModel); effective != "" {
		body = spliceFrontmatterModel(body, effective)
	}
	return body, nil
}

func spliceSquadModel(projectRoot, squadModel string) error {
	path := filepath.Join(OpencodeAgentsDir(projectRoot), "squad.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		data, err = fs.ReadFile(TemplateFS(), "opencode/agents/squad.md")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
	}
	next := spliceFrontmatterModel(string(data), squadModel)
	return os.WriteFile(path, []byte(next), 0o644)
}

func spliceFrontmatterModel(content, model string) string {
	model = strings.TrimSpace(model)
	lines := strings.Split(content, "\n")
	start, end := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(strings.TrimSuffix(line, "\r")) != "---" {
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		end = i
		break
	}
	if start < 0 || end < 0 {
		return content
	}

	fm := append([]string(nil), lines[start+1:end]...)
	modelIdx, modeIdx, descIdx := -1, -1, -1
	for i, line := range fm {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		switch {
		case strings.HasPrefix(trim, "model:"):
			modelIdx = i
		case strings.HasPrefix(trim, "mode:"):
			modeIdx = i
		case strings.HasPrefix(trim, "description:"):
			descIdx = i
		}
	}

	if model == "" {
		if modelIdx < 0 {
			return content
		}
		fm = append(fm[:modelIdx], fm[modelIdx+1:]...)
	} else {
		newLine := "model: " + model
		if modelIdx >= 0 {
			fm[modelIdx] = newLine
		} else {
			at := 0
			if modeIdx >= 0 {
				at = modeIdx + 1
			} else if descIdx >= 0 {
				at = descIdx + 1
			}
			next := make([]string, 0, len(fm)+1)
			next = append(next, fm[:at]...)
			next = append(next, newLine)
			next = append(next, fm[at:]...)
			fm = next
		}
	}

	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:start+1]...)
	out = append(out, fm...)
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n")
}
