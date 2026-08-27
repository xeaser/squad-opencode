package traces

import "time"

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
