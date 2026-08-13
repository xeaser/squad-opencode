package opencodeclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/sst/opencode-sdk-go"
)

// RunRequest is a non-interactive prompt.
type RunRequest struct {
	Directory string
	Agent     string
	Prompt    string
	Title     string
}

// RunResult is the assistant text from a session.
type RunResult struct {
	SessionID string
	Text      string
}

// Runner creates a session and sends a prompt.
type Runner interface {
	Run(ctx context.Context, req RunRequest) (RunResult, error)
}

// SDKRunner uses opencode-sdk-go against a running server.
type SDKRunner struct {
	BaseURL string
}

// Run implements Runner.
func (r SDKRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	if req.Agent == "" {
		req.Agent = "squad"
	}
	if req.Prompt == "" {
		return RunResult{}, fmt.Errorf("prompt is required")
	}
	client := NewClient(r.BaseURL)
	title := req.Title
	if title == "" {
		title = "squad-oc run"
	}
	params := opencode.SessionNewParams{
		Title: opencode.F(title),
	}
	if req.Directory != "" {
		params.Directory = opencode.F(req.Directory)
	}
	sess, err := client.Session.New(ctx, params)
	if err != nil {
		return RunResult{}, fmt.Errorf("create session: %w", err)
	}
	prompt := opencode.SessionPromptParams{
		Agent: opencode.F(req.Agent),
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.SessionPromptParamsPart{
				Type: opencode.F(opencode.SessionPromptParamsPartsTypeText),
				Text: opencode.F(req.Prompt),
			},
		}),
	}
	if req.Directory != "" {
		prompt.Directory = opencode.F(req.Directory)
	}
	resp, err := client.Session.Prompt(ctx, sess.ID, prompt)
	if err != nil {
		return RunResult{}, fmt.Errorf("prompt: %w", err)
	}
	var b strings.Builder
	if resp != nil {
		for _, p := range resp.Parts {
			if p.Text != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(p.Text)
			}
		}
	}
	return RunResult{SessionID: sess.ID, Text: b.String()}, nil
}

// FakeRunner records calls for tests.
type FakeRunner struct {
	Calls []RunRequest
	Text  string
	Err   error
}

// Run implements Runner.
func (f *FakeRunner) Run(_ context.Context, req RunRequest) (RunResult, error) {
	f.Calls = append(f.Calls, req)
	if f.Err != nil {
		return RunResult{}, f.Err
	}
	text := f.Text
	if text == "" {
		text = "ok: " + req.Prompt
	}
	return RunResult{SessionID: "fake-session", Text: text}, nil
}
