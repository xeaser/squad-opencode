package opencodeclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

// DefaultBaseURL is the typical local OpenCode server address.
const DefaultBaseURL = "http://127.0.0.1:4096"

// StartHint is how to expose the HTTP API that squad-oc run/watch talk to.
// The TUI (`opencode`) does not listen on :4096; `opencode serve` does.
const StartHint = "opencode serve"

// StartHelp is the user-facing explanation when the API is down.
const StartHelp = "The TUI (`opencode`) is not the HTTP API. In another terminal, from this project:\n  opencode serve\nThen retry `squad-oc run`. Default URL: " + DefaultBaseURL

// ProbeResult is a soft connectivity check for doctor.
type ProbeResult struct {
	Reachable bool
	BaseURL   string
	Detail    string
	Err       error
}

// NewClient returns an OpenCode REST client pointing at baseURL.
// Empty baseURL uses DefaultBaseURL.
func NewClient(baseURL string) *opencode.Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return opencode.NewClient(
		option.WithBaseURL(baseURL),
	)
}

// ProbeServer tries a lightweight request against the OpenCode server.
// Does not start OpenCode; used only when a server may already be running.
func ProbeServer(ctx context.Context, baseURL string) ProbeResult {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	res := ProbeResult{BaseURL: baseURL}

	// Fast HTTP check first so we fail quickly when nothing is listening.
	httpClient := &http.Client{Timeout: 800 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stringsTrimSlash(baseURL)+"/global/health", nil)
	if err != nil {
		// fall through to SDK
	} else {
		resp, err := httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				res.Reachable = true
				res.Detail = fmt.Sprintf("server responded HTTP %d at %s", resp.StatusCode, baseURL)
				return res
			}
		}
	}

	// Fallback: session list via SDK (works even if health path differs).
	client := NewClient(baseURL)
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	_, err = client.Session.List(ctx, opencode.SessionListParams{})
	if err != nil {
		res.Reachable = false
		res.Detail = fmt.Sprintf("not reachable at %s (%v)", baseURL, err)
		res.Err = err
		return res
	}
	res.Reachable = true
	res.Detail = fmt.Sprintf("session API ok at %s", baseURL)
	return res
}

func stringsTrimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
