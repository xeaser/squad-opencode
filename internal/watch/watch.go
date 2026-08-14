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
	ProjectRoot string
	Execute     bool
	Interval    time.Duration
	Once        bool
	Runner      opencodeclient.Runner
	Lister      IssueLister
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

// Pass runs one poll cycle. Returns whether execute ran.
func Pass(ctx context.Context, opts Options) (executed bool, summary string, err error) {
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

// Loop polls until stop sentinel, context cancel, or Once.
func Loop(ctx context.Context, opts Options) error {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Minute
	}
	for {
		if _, err := os.Stat(StopPath(opts.ProjectRoot)); err == nil {
			return nil
		}
		_, _, err := Pass(ctx, opts)
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
