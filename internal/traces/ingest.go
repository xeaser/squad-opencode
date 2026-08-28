package traces

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xeaser/squad-opencode/internal/opencodestore"
	"github.com/xeaser/squad-opencode/internal/squad"
)

// Ingest reads OpenCode SQLite and appends new turns. n is the number of turns written
// to JSONL. OTLP push failures still count toward n (JSONL is kept) and are returned
// after the batch so callers can log and continue.
func Ingest(projectRoot string, cfg *squad.Config, getenv func(string) string, push func(context.Context, Settings, Span, *Span) error) (int, error) {
	path, explicit, err := opencodestore.ResolveDBPath(projectRoot, cfg, getenv)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) && !explicit {
			return 0, nil
		}
		return 0, err
	}
	db, err := opencodestore.OpenReadOnly(path)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	turns, err := opencodestore.ListTurns(db, projectRoot)
	if err != nil {
		return 0, err
	}
	existing, err := List(projectRoot, 0)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	for _, s := range existing {
		if s.MessageID != "" {
			seen[s.MessageID] = true
		}
	}
	legacy := legacySkipFirst(existing, turns)
	s, err := ResolveSettings(cfg, getenv)
	if err != nil {
		s = Settings{}
	}
	n := 0
	var pushErr error
	for _, t := range turns {
		if t.MessageID == "" || seen[t.MessageID] {
			continue
		}
		if legacy[t.SessionID] == t.MessageID {
			continue
		}
		attrs := map[string]string{"source": "sqlite", "message_id": t.MessageID}
		if t.ParentMessageID != "" {
			attrs["parent_message_id"] = t.ParentMessageID
		}
		if t.ParentSessionID != "" {
			attrs["parent_session_id"] = t.ParentSessionID
		}
		in := RecordInput{
			ParentName:       NameSession,
			Start:            t.UserStart,
			End:              t.AssistantEnd,
			Agent:            t.Agent,
			Prompt:           t.Prompt,
			Completion:       t.Completion,
			SessionID:        t.SessionID,
			MessageID:        t.MessageID,
			Attrs:            attrs,
			HasGeneration:    true,
			Provider:         t.Provider,
			Model:            t.Model,
			InputTokens:      t.InputTokens,
			OutputTokens:     t.OutputTokens,
			ReasoningTokens:  t.ReasoningTokens,
			CacheReadTokens:  t.CacheReadTokens,
			CacheWriteTokens: t.CacheWriteTokens,
			Cost:             t.Cost,
		}
		if t.Err {
			in.Err = fmt.Errorf("opencode message error")
		}
		parent, child := Build(in)
		// gen_ai.chat starts when the assistant message is created, not the user turn.
		if child != nil && !t.AssistantStart.IsZero() {
			child.Start = t.AssistantStart
		}
		if err := writeBuilt(projectRoot, parent, child, s, push); err != nil {
			if IsOTLPPushError(err) {
				// JSONL already written; keep going and surface push once.
				seen[t.MessageID] = true
				n++
				if pushErr == nil {
					pushErr = err
				}
				continue
			}
			return n, err
		}
		seen[t.MessageID] = true
		n++
	}
	return n, pushErr
}

// IsOTLPPushError reports whether err is from Write's OTLP export step
// (JSONL append already succeeded).
func IsOTLPPushError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "otlp push:")
}

// legacySkipFirst skips the earliest completed assistant when a session has a
// squad-oc.run / squad-oc.watch.execute span with empty MessageID (CLI parent
// already covered that turn). Durable across later ingests: once such a CLI
// parent exists, the earliest assistant stays skipped even after other
// messageIds appear in JSONL.
func legacySkipFirst(spans []Span, turns []opencodestore.Turn) map[string]string {
	hasCLIEmpty := map[string]bool{}
	for _, s := range spans {
		if s.SessionID == "" {
			continue
		}
		if (s.Name == "squad-oc.run" || s.Name == "squad-oc.watch.execute") && s.MessageID == "" {
			hasCLIEmpty[s.SessionID] = true
		}
	}
	skip := map[string]string{}
	for _, t := range turns {
		if !hasCLIEmpty[t.SessionID] {
			continue
		}
		if _, ok := skip[t.SessionID]; !ok {
			skip[t.SessionID] = t.MessageID
		}
	}
	return skip
}
