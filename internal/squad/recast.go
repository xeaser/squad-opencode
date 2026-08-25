package squad

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

func memberID(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

func appendMemberRow(content, name, role, id, model string) (string, error) {
	row := fmt.Sprintf("| %s | %s | `.squad/agents/%s/charter.md` | Active |", name, role, id)
	if model != "" {
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
	block := "\n## Members\n\n| Name | Role | Charter | Status |\n|------|------|---------|--------|\n" + row + "\n"
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
