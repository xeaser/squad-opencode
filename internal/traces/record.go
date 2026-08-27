package traces

import (
	"context"
	"fmt"
	"time"
)

// RecordInput is the shared parent+child builder used by run and watch.
type RecordInput struct {
	ParentName string
	Start, End time.Time
	Err        error
	Agent      string
	Prompt     string
	Completion string
	SessionID  string
	Attrs      map[string]string
	// Generation set when Session.Prompt returned (even if Info empty).
	HasGeneration                                                                 bool
	Provider                                                                      string
	Model                                                                         string
	InputTokens, OutputTokens, ReasoningTokens, CacheReadTokens, CacheWriteTokens int
	Cost                                                                          float64
}

// Build constructs the parent span and optional gen_ai.chat child. It does not write JSONL.
func Build(in RecordInput) (parent Span, child *Span) {
	agent := in.Agent
	if agent == "" {
		agent = "squad"
	}
	status := "OK"
	if in.Err != nil {
		status = "ERROR"
	}
	attrs := make(map[string]string, len(in.Attrs)+1)
	for k, v := range in.Attrs {
		attrs[k] = v
	}
	if _, ok := attrs["agent"]; !ok {
		attrs["agent"] = agent
	}

	parent = Span{
		Name:       in.ParentName,
		Start:      in.Start,
		End:        in.End,
		Status:     status,
		Attributes: attrs,
		SessionID:  in.SessionID,
		Agent:      agent,
	}
	if parent.TraceID == "" {
		if id, err := newHex(16); err == nil {
			parent.TraceID = id
		}
	}
	if parent.SpanID == "" {
		if id, err := newHex(8); err == nil {
			parent.SpanID = id
		}
	}
	if !in.HasGeneration {
		return parent, nil
	}
	cid, _ := newHex(8)
	c := Span{
		Name:             NameChat,
		TraceID:          parent.TraceID,
		SpanID:           cid,
		ParentID:         parent.SpanID,
		Start:            in.Start,
		End:              in.End,
		Status:           "OK",
		SessionID:        in.SessionID,
		Agent:            agent,
		Provider:         in.Provider,
		Model:            in.Model,
		InputTokens:      in.InputTokens,
		OutputTokens:     in.OutputTokens,
		ReasoningTokens:  in.ReasoningTokens,
		CacheReadTokens:  in.CacheReadTokens,
		CacheWriteTokens: in.CacheWriteTokens,
		Cost:             in.Cost,
		Prompt:           in.Prompt,
		Completion:       in.Completion,
	}
	return parent, &c
}

// Write appends parent (and child if non-nil) when projectRoot != "".
// If s.Endpoint != "" it then calls push (default Push). Errors are wrapped
// as "append: …" or "otlp push: …" so callers can log one stderr line.
func Write(projectRoot string, in RecordInput, s Settings, push func(context.Context, Settings, Span, *Span) error) error {
	parent, child := Build(in)
	if projectRoot != "" {
		if err := Append(projectRoot, parent); err != nil {
			return fmt.Errorf("append: %w", err)
		}
		if child != nil {
			if err := Append(projectRoot, *child); err != nil {
				return fmt.Errorf("append: %w", err)
			}
		}
	}
	if s.Endpoint == "" {
		return nil
	}
	if push == nil {
		push = Push
	}
	if err := push(context.Background(), s, parent, child); err != nil {
		return fmt.Errorf("otlp push: %w", err)
	}
	return nil
}
