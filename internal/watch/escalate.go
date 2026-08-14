package watch

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// Escalator runs one recovery step after a failed Pass.
type Escalator interface {
	Reset(ctx context.Context) error      // tier 1
	ReprobeAuth(ctx context.Context) error // tier 2 — `gh auth status`
	GitPull(ctx context.Context) error     // tier 3
}

// DefaultEscalator runs recovery commands in Dir (project root).
type DefaultEscalator struct {
	Dir string
}

// Reset is a no-op (tier 1: log and continue).
func (DefaultEscalator) Reset(context.Context) error { return nil }

// ReprobeAuth runs `gh auth status` (output ignored).
func (d DefaultEscalator) ReprobeAuth(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	if d.Dir != "" {
		cmd.Dir = d.Dir
	}
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "GH_FORCE_TTY=0", "GH_PROMPT_DISABLED=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	return cmd.Run()
}

// GitPull runs `git pull --ff-only`.
func (d DefaultEscalator) GitPull(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "pull", "--ff-only")
	if d.Dir != "" {
		cmd.Dir = d.Dir
	}
	return cmd.Run()
}

// NextTier maps consecutive failures to 1..4.
func NextTier(consecutive int) int {
	if consecutive < 1 {
		return 1
	}
	if consecutive > 4 {
		return 4
	}
	return consecutive
}

func applyEscalation(ctx context.Context, opts Options, consecutive int) {
	esc := opts.Escalator
	if esc == nil {
		esc = DefaultEscalator{Dir: opts.ProjectRoot}
	}
	switch NextTier(consecutive) {
	case 1:
		notify(opts, NotifyImportant, "escalate: reset")
		_ = esc.Reset(ctx)
	case 2:
		if opts.Verbose {
			notify(opts, NotifyAll, "auth probe: gh auth status")
		}
		if err := esc.ReprobeAuth(ctx); err != nil {
			notify(opts, NotifyImportant, "auth probe failed: "+err.Error())
		} else {
			notify(opts, NotifyImportant, "auth probe ok")
		}
	case 3:
		if err := esc.GitPull(ctx); err != nil {
			notify(opts, NotifyImportant, "git pull failed: "+err.Error())
		} else {
			notify(opts, NotifyImportant, "git pull ok")
		}
	default:
		d := opts.Backoff
		if d <= 0 {
			d = 30 * time.Minute
		}
		notify(opts, NotifyImportant, "escalate: backoff "+d.String())
		if opts.Sleep != nil {
			opts.Sleep(d)
			return
		}
		select {
		case <-ctx.Done():
		case <-time.After(d):
		}
	}
}