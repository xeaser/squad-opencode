package opencodeclient

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EnvBaseURL overrides the OpenCode HTTP API address.
const EnvBaseURL = "OPENCODE_BASE_URL"

// ProbeFn and StartFn are the probe / spawn hooks (tests replace them).
var (
	ProbeFn = ProbeServer
	StartFn = startServe
)

// EnsureResult is what run/watch print after making sure the API is up.
type EnsureResult struct {
	BaseURL  string
	Attached bool
	Started  bool
	Message  string
}

// ResolveBaseURL prefers --url, then OPENCODE_BASE_URL, then DefaultBaseURL.
func ResolveBaseURL(explicit string) string {
	if s := strings.TrimSpace(explicit); s != "" {
		return normalizeURL(s)
	}
	if s := strings.TrimSpace(os.Getenv(EnvBaseURL)); s != "" {
		return normalizeURL(s)
	}
	return DefaultBaseURL
}

// IsDefaultLocal is true only for http://127.0.0.1:4096 and http://localhost:4096.
// Auto-start is refused for any other host or port.
func IsDefaultLocal(raw string) bool {
	u, err := url.Parse(normalizeURL(raw))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host != "127.0.0.1" && host != "localhost" {
		return false
	}
	port := u.Port()
	return port == "4096"
}

func normalizeURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimRight(s, "/")
	if s != "" && !strings.Contains(s, "://") {
		s = "http://" + s
	}
	return s
}

// EnsureAPI attaches to an already-running server, or starts `opencode serve`
// only for the default local URL.
func EnsureAPI(ctx context.Context, explicitURL, projectRoot string) (EnsureResult, error) {
	base := ResolveBaseURL(explicitURL)
	res := EnsureResult{BaseURL: base}
	if ProbeFn(ctx, base).Reachable {
		res.Attached = true
		res.Message = "attached to " + base
		return res, nil
	}
	if !IsDefaultLocal(base) {
		return res, fmt.Errorf("OpenCode HTTP API not reachable at %s\nAuto-start only for %s. Start the server yourself, or unset %s / --url.\n\n%s",
			base, DefaultBaseURL, EnvBaseURL, StartHelp)
	}
	if err := StartFn(projectRoot); err != nil {
		return res, fmt.Errorf("start %s: %w\n%s", StartHint, err, StartHelp)
	}
	res.Started = true
	if err := waitReady(ctx, base); err != nil {
		return res, err
	}
	res.Message = "started opencode serve at " + base
	return res, nil
}

func waitReady(ctx context.Context, base string) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if ProbeFn(ctx, base).Reachable {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("started opencode serve but %s never became ready", base)
}

func startServe(projectRoot string) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not on PATH")
	}
	cmd := exec.Command(bin, "serve", "--hostname", "127.0.0.1", "--port", "4096")
	if projectRoot != "" {
		cmd.Dir = projectRoot
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Intentionally not Wait()'ing: serve outlives this process.
	go func() { _ = cmd.Wait() }()
	return nil
}
