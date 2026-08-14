package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/squad"
)

// Issue is a work item (usually a GitHub issue).
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// IssueLister lists actionable issues.
type IssueLister interface {
	List(ctx context.Context) ([]Issue, error)
}

// StaticLister returns a fixed list (tests).
type StaticLister struct {
	Issues []Issue
	Err    error
}

// List implements IssueLister.
func (s StaticLister) List(context.Context) ([]Issue, error) {
	return s.Issues, s.Err
}

// Options control a watch pass.
type Options struct {
	ProjectRoot    string
	Execute        bool
	Interval       time.Duration
	Once           bool
	OvernightStart string // HH:MM local, e.g. "18:00"
	OvernightEnd   string // HH:MM local, e.g. "08:00"
	Now            func() time.Time
	Runner         opencodeclient.Runner
	Lister         IssueLister
}

// StopPath is the graceful-stop sentinel.
func StopPath(projectRoot string) string {
	return filepath.Join(squad.ResolveDir(projectRoot), "ralph-stop")
}

// BuildContext renders a prompt from team state + issues.
func BuildContext(projectRoot string, issues []Issue) (string, error) {
	var b strings.Builder
	b.WriteString("# Squad watch context\n\n")
	members, err := squad.ReadTeam(projectRoot)
	if err != nil {
		return "", err
	}
	b.WriteString("## Team\n")
	for _, m := range members {
		fmt.Fprintf(&b, "- %s (%s) [%s]\n", m.Name, m.Role, m.Status)
	}
	b.WriteString("\n## Issues\n")
	if len(issues) == 0 {
		b.WriteString("(none)\n")
	}
	for _, is := range issues {
		fmt.Fprintf(&b, "- #%d %s (%s)\n", is.Number, is.Title, is.State)
	}
	dec := filepath.Join(squad.ResolveDir(projectRoot), "decisions.md")
	if data, err := os.ReadFile(dec); err == nil {
		b.WriteString("\n## Decisions (excerpt)\n\n")
		s := string(data)
		if len(s) > 1500 {
			s = s[:1500] + "\n…"
		}
		b.WriteString(s)
	}
	b.WriteString("\n\nPick the highest-priority issue you can complete. Escalate blockers to the human.\n")
	return b.String(), nil
}

// InOvernight reports whether now falls in the [start, end) quiet window.
// A window with start > end crosses midnight (18:00–08:00).
func InOvernight(now time.Time, start, end string) (bool, error) {
	start, end = strings.TrimSpace(start), strings.TrimSpace(end)
	if start == "" && end == "" {
		return false, nil
	}
	if start == "" || end == "" {
		return false, fmt.Errorf("overnight needs both start and end (HH:MM)")
	}
	s, err := parseClock(start)
	if err != nil {
		return false, err
	}
	e, err := parseClock(end)
	if err != nil {
		return false, err
	}
	if s == e {
		return false, nil
	}
	n := now.Hour()*60 + now.Minute()
	if s < e {
		return n >= s && n < e, nil
	}
	return n >= s || n < e, nil
}

func parseClock(s string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid time %q (want HH:MM)", s)
	}
	return h*60 + m, nil
}

func clockNow(opts Options) time.Time {
	if opts.Now != nil {
		return opts.Now()
	}
	return time.Now()
}

// Pass runs one poll cycle. Returns whether execute ran.
func Pass(ctx context.Context, opts Options) (executed bool, summary string, err error) {
	defer func() { writePassHealth(opts, summary, err) }()
	quiet, err := InOvernight(clockNow(opts), opts.OvernightStart, opts.OvernightEnd)
	if err != nil {
		return false, "", err
	}
	if quiet {
		return false, "overnight quiet until " + opts.OvernightEnd, nil
	}
	if opts.Lister == nil {
		return false, "", fmt.Errorf("no issue lister")
	}
	issues, err := opts.Lister.List(ctx)
	if err != nil {
		return false, "", err
	}
	ctxText, err := BuildContext(opts.ProjectRoot, issues)
	if err != nil {
		return false, "", err
	}
	summary = fmt.Sprintf("issues=%d execute=%v", len(issues), opts.Execute)
	if !opts.Execute {
		return false, summary + "\n" + ctxText, nil
	}
	if opts.Runner == nil {
		return false, "", fmt.Errorf("execute requires a runner")
	}
	res, err := opts.Runner.Run(ctx, opencodeclient.RunRequest{
		Directory: opts.ProjectRoot,
		Agent:     "squad",
		Prompt:    ctxText,
		Title:     "squad-oc watch",
	})
	if err != nil {
		return false, summary, err
	}
	return true, summary + "\n" + res.Text, nil
}

func writePassHealth(opts Options, summary string, err error) {
	if opts.ProjectRoot == "" {
		return
	}
	now := clockNow(opts)
	h, rerr := ReadHealth(opts.ProjectRoot)
	if rerr != nil && !os.IsNotExist(rerr) {
		return
	}
	h.LastPoll = now
	h.LastSummary = firstLine(summary)
	h.Round++
	if err != nil {
		h.LastError = err.Error()
		h.Consecutive++
	} else {
		h.LastError = ""
		h.Consecutive = 0
	}
	quiet, _ := InOvernight(now, opts.OvernightStart, opts.OvernightEnd)
	h.Overnight = quiet
	_ = WriteHealth(opts.ProjectRoot, h)
}

// Loop polls until stop sentinel, context cancel, or Once.
func Loop(ctx context.Context, opts Options) error {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Minute
	}
	now := clockNow(opts)
	h, err := ReadHealth(opts.ProjectRoot)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	h.PID = os.Getpid()
	h.StartedAt = now
	if err := WriteHealth(opts.ProjectRoot, h); err != nil {
		return err
	}
	for {
		if _, err := os.Stat(StopPath(opts.ProjectRoot)); err == nil {
			return nil
		}
		_, _, err := Pass(ctx, opts)
		now = clockNow(opts)
		h, rerr := ReadHealth(opts.ProjectRoot)
		if rerr != nil && !os.IsNotExist(rerr) {
			return rerr
		}
		h.PID = os.Getpid()
		if h.StartedAt.IsZero() {
			h.StartedAt = now
		}
		h.NextPoll = now.Add(opts.Interval)
		if werr := WriteHealth(opts.ProjectRoot, h); werr != nil {
			return werr
		}
		if err != nil {
			return err
		}
		if opts.Once {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
}
