package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/traces"
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
	Labels         []string
	LogFile        string
	Verbose        bool
	Notify         NotifyLevel
	Logger         func(level NotifyLevel, msg string)
	Escalator      Escalator
	Backoff        time.Duration
	Sleep          func(time.Duration)
	Backend        StateBackend
}

func loadHealth(ctx context.Context, opts Options) (Health, error) {
	if opts.Backend != nil {
		return opts.Backend.Load(ctx)
	}
	if opts.ProjectRoot == "" {
		return Health{}, nil
	}
	h, err := ReadHealth(opts.ProjectRoot)
	if err != nil && os.IsNotExist(err) {
		return Health{}, nil
	}
	return h, err
}

func saveHealth(ctx context.Context, opts Options, h Health) error {
	if opts.Backend != nil {
		if err := opts.Backend.Save(ctx, h); err != nil {
			return err
		}
	}
	if opts.ProjectRoot == "" {
		return nil
	}
	return WriteHealth(opts.ProjectRoot, h)
}

// NotifyLevel selects which watch lines are emitted.
type NotifyLevel int

const (
	NotifyAll NotifyLevel = iota
	NotifyImportant
	NotifyNone
)

// ParseNotifyLevel maps all|important|none (default important).
func ParseNotifyLevel(s string) (NotifyLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "important":
		return NotifyImportant, nil
	case "all":
		return NotifyAll, nil
	case "none":
		return NotifyNone, nil
	default:
		return NotifyImportant, fmt.Errorf("invalid --notify-level %q (want all|important|none)", s)
	}
}

func passesNotify(opts Options, level NotifyLevel) bool {
	switch opts.Notify {
	case NotifyNone:
		return opts.Verbose && level == NotifyAll
	case NotifyAll:
		return true
	default:
		return level == NotifyImportant
	}
}

func notify(opts Options, level NotifyLevel, msg string) {
	if !passesNotify(opts, level) {
		return
	}
	if opts.Logger != nil {
		opts.Logger(level, msg)
	}
	if opts.LogFile != "" {
		_ = appendLog(opts.LogFile, msg)
	}
}

func appendLog(path, msg string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	_, err = f.WriteString(msg)
	return err
}

// StopPath is the graceful-stop sentinel.
func StopPath(projectRoot string) string {
	return filepath.Join(squad.ResolveDir(projectRoot), "ralph-stop")
}

// BuildContext renders a prompt from team state + issues.
// Optional labels are listed as a filter (gh already returns open issues).
func BuildContext(projectRoot string, issues []Issue, labels ...string) (string, error) {
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
	if len(labels) > 0 {
		b.WriteString("\n## Label filter\n")
		for _, label := range labels {
			fmt.Fprintf(&b, "- %s\n", label)
		}
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
	prevOvernight := false
	if h, rerr := loadHealth(context.Background(), opts); rerr == nil {
		prevOvernight = h.Overnight
	}
	quiet, err := InOvernight(clockNow(opts), opts.OvernightStart, opts.OvernightEnd)
	if opts.Verbose {
		notify(opts, NotifyAll, fmt.Sprintf("overnight check: quiet=%v", quiet))
	}
	if err != nil {
		notify(opts, NotifyImportant, "overnight error: "+err.Error())
		return false, "", err
	}
	if quiet {
		summary = "overnight quiet until " + opts.OvernightEnd
		if !prevOvernight {
			notify(opts, NotifyImportant, "overnight enter until "+opts.OvernightEnd)
		}
		return false, summary, nil
	}
	if prevOvernight {
		notify(opts, NotifyImportant, "overnight exit")
	}
	if opts.Lister == nil {
		err = fmt.Errorf("no issue lister")
		notify(opts, NotifyImportant, err.Error())
		return false, "", err
	}
	issues, err := opts.Lister.List(ctx)
	if err != nil {
		notify(opts, NotifyImportant, "list error: "+err.Error())
		return false, "", err
	}
	ctxText, err := BuildContext(opts.ProjectRoot, issues, opts.Labels...)
	if err != nil {
		notify(opts, NotifyImportant, "context error: "+err.Error())
		return false, "", err
	}
	summary = fmt.Sprintf("issues=%d execute=%v", len(issues), opts.Execute)
	if !opts.Execute {
		if opts.Notify == NotifyAll {
			notify(opts, NotifyAll, ctxText)
		}
		return false, summary + "\n" + ctxText, nil
	}
	if opts.Runner == nil {
		err = fmt.Errorf("execute requires a runner")
		notify(opts, NotifyImportant, err.Error())
		return false, "", err
	}
	notify(opts, NotifyImportant, "execute started")
	start := time.Now()
	res, err := opts.Runner.Run(ctx, opencodeclient.RunRequest{
		Directory: opts.ProjectRoot,
		Agent:     "squad",
		Prompt:    ctxText,
		Title:     "squad-oc watch",
	})
	status := "OK"
	if err != nil {
		status = "ERROR"
	}
	if opts.ProjectRoot != "" {
		_ = traces.Append(opts.ProjectRoot, traces.Span{
			Name:   "squad-oc.watch.execute",
			Start:  start,
			End:    time.Now(),
			Status: status,
			Attributes: map[string]string{
				"issues": strconv.Itoa(len(issues)),
			},
		})
	}
	if err != nil {
		notify(opts, NotifyImportant, "execute error: "+err.Error())
		return false, summary, err
	}
	notify(opts, NotifyImportant, "execute finished")
	return true, summary + "\n" + res.Text, nil
}

func writePassHealth(opts Options, summary string, err error) {
	if opts.ProjectRoot == "" && opts.Backend == nil {
		return
	}
	now := clockNow(opts)
	h, rerr := loadHealth(context.Background(), opts)
	if rerr != nil {
		return
	}
	if h.StartedAt.IsZero() {
		h.StartedAt = now
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
	_ = saveHealth(context.Background(), opts, h)
}

// Loop polls until stop sentinel, context cancel, or Once.
func Loop(ctx context.Context, opts Options) error {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Minute
	}
	now := clockNow(opts)
	h, err := loadHealth(ctx, opts)
	if err != nil {
		return err
	}
	h.PID = os.Getpid()
	h.StartedAt = now
	if err := saveHealth(ctx, opts, h); err != nil {
		return err
	}
	for {
		if _, err := os.Stat(StopPath(opts.ProjectRoot)); err == nil {
			notify(opts, NotifyImportant, "stop sentinel")
			return nil
		}
		_, _, err := Pass(ctx, opts)
		now = clockNow(opts)
		h, rerr := loadHealth(ctx, opts)
		if rerr != nil {
			return rerr
		}
		h.PID = os.Getpid()
		if h.StartedAt.IsZero() {
			h.StartedAt = now
		}
		h.NextPoll = now.Add(opts.Interval)
		if werr := saveHealth(ctx, opts, h); werr != nil {
			return werr
		}
		if err != nil {
			if opts.Once {
				return err
			}
			tier := NextTier(h.Consecutive)
			applyEscalation(ctx, opts, h.Consecutive)
			if tier >= 4 {
				h.Consecutive = 0
				if werr := saveHealth(ctx, opts, h); werr != nil {
					return werr
				}
			}
		}
		if opts.Once {
			return nil
		}
		if opts.Verbose {
			notify(opts, NotifyAll, "sleep "+opts.Interval.String())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
}
