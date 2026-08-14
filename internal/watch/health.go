package watch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// Health is the live watch snapshot written to ralph-status.json (and optional state backends).
type Health struct {
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"startedAt"`
	LastPoll    time.Time `json:"lastPoll"`
	LastSummary string    `json:"lastSummary"`
	LastError   string    `json:"lastError,omitempty"`
	Consecutive int       `json:"consecutiveErrors"`
	NextPoll    time.Time `json:"nextPoll"`
	Round       int       `json:"round"`
	Overnight   bool      `json:"overnight"`
}

// StatusPath is ralph-status.json under the live team directory.
func StatusPath(projectRoot string) string {
	return filepath.Join(squad.ResolveDir(projectRoot), "ralph-status.json")
}

// WriteHealth writes Health as JSON to StatusPath.
func WriteHealth(projectRoot string, h Health) error {
	path := StatusPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// ReadHealth loads Health from StatusPath.
func ReadHealth(projectRoot string) (Health, error) {
	var h Health
	data, err := os.ReadFile(StatusPath(projectRoot))
	if err != nil {
		return h, err
	}
	if err := json.Unmarshal(data, &h); err != nil {
		return Health{}, err
	}
	return h, nil
}

// FormatHealth renders a one-shot --health report.
func FormatHealth(h Health, now time.Time) string {
	var b strings.Builder
	b.WriteString("Ralph watch\n\n")
	if h.PID == 0 {
		b.WriteString("PID: n/a\n")
	} else {
		fmt.Fprintf(&b, "PID: %d\n", h.PID)
	}
	if !h.StartedAt.IsZero() {
		fmt.Fprintf(&b, "Uptime: %s\n", formatUptime(now.Sub(h.StartedAt)))
	}
	if h.LastPoll.IsZero() {
		b.WriteString("Last poll: never\n")
	} else {
		fmt.Fprintf(&b, "Last poll: %s\n", formatAgo(now.Sub(h.LastPoll)))
	}
	fmt.Fprintf(&b, "Last: %s\n", firstLine(h.LastSummary))
	if h.NextPoll.IsZero() {
		b.WriteString("Next poll: not scheduled\n")
	} else {
		fmt.Fprintf(&b, "Next poll: %s (%s)\n", h.NextPoll.Format("15:04"), formatIn(h.NextPoll.Sub(now)))
	}
	fmt.Fprintf(&b, "Round: %d\n", h.Round)
	return b.String()
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func formatUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatAgo(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return "just now"
	}
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("%d %s ago", mins, plural(mins, "minute"))
	}
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%d %s ago", hours, plural(hours, "hour"))
	}
	days := hours / 24
	return fmt.Sprintf("%d %s ago", days, plural(days, "day"))
}

func formatIn(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	if d < time.Minute {
		return "in less than a minute"
	}
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("in %d %s", mins, plural(mins, "minute"))
	}
	hours := int(d.Hours())
	return fmt.Sprintf("in %d %s", hours, plural(hours, "hour"))
}

func plural(n int, unit string) string {
	if n == 1 {
		return unit
	}
	return unit + "s"
}
