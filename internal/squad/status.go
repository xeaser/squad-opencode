package squad

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StatusReport builds the human-readable squad status for root.
func StatusReport(root string) (string, error) {
	if !IsInitialized(root) {
		return "", fmt.Errorf("not initialized")
	}
	det := Detect(root)
	members, err := ReadTeam(root)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Squad status — %s\n", root)
	if det.Config != nil {
		fmt.Fprintf(&b, "Host: %s  Preset: %s\n", det.Config.Host, det.Config.Preset)
		if det.Config.ProjectDescription != "" {
			fmt.Fprintf(&b, "Project: %s\n", det.Config.ProjectDescription)
		}
		if det.Config.ExternalPath != "" {
			fmt.Fprintf(&b, "External: %s\n", det.Config.ExternalPath)
		}
		if det.Config.LinkPath != "" {
			fmt.Fprintf(&b, "Link: %s\n", det.Config.LinkPath)
		}
		if det.Config.LinkURL != "" {
			fmt.Fprintf(&b, "Link URL: %s\n", det.Config.LinkURL)
			sha := det.Config.LinkSHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			if sha != "" {
				if det.Config.LinkRef != "" {
					fmt.Fprintf(&b, "Link rev: %s @ %s\n", det.Config.LinkRef, sha)
				} else {
					fmt.Fprintf(&b, "Link rev: %s\n", sha)
				}
			}
		}
	}
	b.WriteByte('\n')
	if len(members) == 0 {
		b.WriteString("(no members parsed from team.md)\n")
	} else {
		width := 4
		for _, m := range members {
			if len(m.Name) > width {
				width = len(m.Name)
			}
		}
		fmt.Fprintf(&b, "%-*s  Role\n%s  ----\n", width, "Name", strings.Repeat("-", width))
		for _, m := range members {
			fmt.Fprintf(&b, "%-*s  %s  [%s]\n", width, m.Name, m.Role, m.Status)
		}
	}

	if excerpt := decisionsExcerpt(root, 8); excerpt != "" {
		b.WriteByte('\n')
		b.WriteString("Decisions\n")
		b.WriteString(excerpt)
		if !strings.HasSuffix(excerpt, "\n") {
			b.WriteByte('\n')
		}
	}

	if watch := watchExcerpt(root); watch != "" {
		b.WriteByte('\n')
		b.WriteString(watch)
		if !strings.HasSuffix(watch, "\n") {
			b.WriteByte('\n')
		}
	}

	return b.String(), nil
}

// decisionsExcerpt returns up to n non-empty lines from decisions.md, skipping the # title.
func decisionsExcerpt(root string, n int) string {
	path := filepath.Join(ResolveDir(root), "decisions.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip markdown H1 title only (e.g. "# Decisions").
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "##") {
			continue
		}
		lines = append(lines, trimmed)
		if len(lines) >= n {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

type ralphStatusFile struct {
	LastPoll    string `json:"lastPoll"`
	LastSummary string `json:"lastSummary"`
}

// watchExcerpt prints lastPoll/lastSummary when ralph-status.json exists.
func watchExcerpt(root string) string {
	path := filepath.Join(ResolveDir(root), "ralph-status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var st ralphStatusFile
	if err := json.Unmarshal(data, &st); err != nil {
		return ""
	}
	if st.LastPoll == "" && st.LastSummary == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("Watch\n")
	if st.LastPoll != "" {
		fmt.Fprintf(&b, "lastPoll: %s\n", st.LastPoll)
	}
	if st.LastSummary != "" {
		fmt.Fprintf(&b, "lastSummary: %s\n", st.LastSummary)
	}
	return b.String()
}
