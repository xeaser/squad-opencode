package opencodeclient

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/traces"
)

// RunRequest is a non-interactive prompt.
type RunRequest struct {
	Directory  string
	Agent      string
	Prompt     string
	Title      string
	SkipRecord bool
}

// RunResult is the assistant text from a session.
type RunResult struct {
	SessionID                                                                     string
	Text                                                                          string
	HasGeneration                                                                 bool
	Provider, Model                                                               string
	InputTokens, OutputTokens, ReasoningTokens, CacheReadTokens, CacheWriteTokens int
	Cost                                                                          float64
}

// pushOTLP is the OTel export hook. Tests replace it with a failing func.
var pushOTLP = traces.Push

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
	start := time.Now()
	res, err := r.run(ctx, req)
	if !req.SkipRecord {
		recordRun(req, start, err, res)
	}
	return res, err
}

func (r SDKRunner) run(ctx context.Context, req RunRequest) (RunResult, error) {
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
	res := RunResult{SessionID: sess.ID, Text: b.String(), HasGeneration: true}
	if resp != nil {
		res.Provider = resp.Info.ProviderID
		res.Model = resp.Info.ModelID
		res.Cost = resp.Info.Cost
		res.InputTokens = int(math.Round(resp.Info.Tokens.Input))
		res.OutputTokens = int(math.Round(resp.Info.Tokens.Output))
		res.ReasoningTokens = int(math.Round(resp.Info.Tokens.Reasoning))
		res.CacheReadTokens = int(math.Round(resp.Info.Tokens.Cache.Read))
		res.CacheWriteTokens = int(math.Round(resp.Info.Tokens.Cache.Write))
		if res.SessionID == "" {
			res.SessionID = resp.Info.SessionID
		}
	}
	return res, nil
}

func recordRun(req RunRequest, start time.Time, runErr error, res RunResult) {
	s, err := traces.ResolveSettings(squad.Detect(req.Directory).Config, os.Getenv)
	if err != nil {
		s = traces.Settings{}
	}
	if err := traces.Write(req.Directory, traces.RecordInput{
		ParentName:       "squad-oc.run",
		Start:            start,
		End:              time.Now(),
		Err:              runErr,
		Agent:            req.Agent,
		Prompt:           req.Prompt,
		Completion:       res.Text,
		SessionID:        res.SessionID,
		Attrs:            map[string]string{"prompt_bytes": strconv.Itoa(len(req.Prompt))},
		HasGeneration:    res.HasGeneration,
		Provider:         res.Provider,
		Model:            res.Model,
		InputTokens:      res.InputTokens,
		OutputTokens:     res.OutputTokens,
		ReasoningTokens:  res.ReasoningTokens,
		CacheReadTokens:  res.CacheReadTokens,
		CacheWriteTokens: res.CacheWriteTokens,
		Cost:             res.Cost,
	}, s, pushOTLP); err != nil {
		fmt.Fprintln(os.Stderr, "traces: otlp push:", err)
	}
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
	return RunResult{SessionID: "fake-session", Text: text, HasGeneration: true}, nil
}
