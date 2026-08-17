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
func AddMember(projectRoot, name, role string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
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
	next, err := appendMemberRow(string(raw), name, role, id)
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
	host := filepath.Join(OpencodeAgentsDir(projectRoot), found.ID+".md")
	if err := os.Remove(host); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func memberID(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

func appendMemberRow(content, name, role, id string) (string, error) {
	row := fmt.Sprintf("| %s | %s | `.squad/agents/%s/charter.md` | Active |", name, role, id)
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

// Recast writes .opencode/agents/<id>.md from the live team. Does not touch squad.md.
// Under the office theme it also writes display-name files (michael.md) so @michael
// works. OpenCode has no agent alias; the extra file is the same body as the id file.
func Recast(projectRoot string) (RecastResult, error) {
	if !IsInitialized(projectRoot) {
		return RecastResult{}, fmt.Errorf("not initialized")
	}
	members, err := ReadTeam(projectRoot)
	if err != nil {
		return RecastResult{}, err
	}
	destDir := OpencodeAgentsDir(projectRoot)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return RecastResult{}, err
	}
	theme := ""
	if det := Detect(projectRoot); det.Config != nil {
		theme = det.Config.Theme
	}
	var res RecastResult
	tpl := TemplateFS()
	kept := map[string]struct{}{}
	for _, m := range members {
		if m.ID == "" || m.ID == "squad" {
			continue
		}
		body, err := agentMarkdown(tpl, m)
		if err != nil {
			return res, err
		}
		if err := writeAgentFile(destDir, m.ID, body, &res, kept); err != nil {
			return res, err
		}
		if theme == ThemeOffice {
			if slug := OfficeMentionSlug(m.ID); slug != "" {
				if err := writeAgentFile(destDir, slug, body, &res, kept); err != nil {
					return res, err
				}
			}
		}
	}
	for _, slug := range officeMentionSlugs() {
		if _, ok := kept[slug]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(destDir, slug+".md"))
	}
	return res, nil
}

func writeAgentFile(destDir, id, body string, res *RecastResult, kept map[string]struct{}) error {
	if _, ok := kept[id]; ok {
		return nil
	}
	if err := os.WriteFile(filepath.Join(destDir, id+".md"), []byte(body), 0o644); err != nil {
		return err
	}
	res.Written++
	res.IDs = append(res.IDs, id)
	kept[id] = struct{}{}
	return nil
}

func agentMarkdown(tpl fs.FS, m TeamMember) (string, error) {
	data, err := fs.ReadFile(tpl, "opencode/agents/"+m.ID+".md")
	if err == nil {
		return string(data), nil
	}
	return fmt.Sprintf(`---
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
`, m.Name, m.Role, m.Name, m.Role, m.ID, m.ID), nil
}
