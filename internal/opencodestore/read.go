package opencodestore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Query strings — tests assert they never name credential tables.
const (
	qSessions = `SELECT id, parent_id, directory, agent, model, time_created, time_updated, time_archived FROM session`
	qMessages = `SELECT id, session_id, time_created, time_updated, data FROM message`
	qParts    = `SELECT message_id, data FROM part`
)

var schemaSkipOnce sync.Once

// Turn is one completed assistant message in a project session.
type Turn struct {
	SessionID, MessageID, ParentMessageID, ParentSessionID string
	Agent, Provider, Model                                 string
	UserStart, AssistantStart, AssistantEnd                time.Time
	InputTokens, OutputTokens, ReasoningTokens             int
	CacheReadTokens, CacheWriteTokens                      int
	Cost                                                   float64
	Prompt, Completion                                     string
	Err                                                    bool
}

// ListTurns returns completed assistant turns for projectRoot.
func ListTurns(db *sql.DB, projectRoot string) ([]Turn, error) {
	sessions, err := loadSessions(db, projectRoot)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	msgs, err := loadMessages(db)
	if err != nil {
		return nil, err
	}
	parts, err := loadParts(db)
	if err != nil {
		return nil, err
	}

	var out []Turn
	for _, m := range msgs {
		if m.role != "assistant" || m.completed.IsZero() {
			continue
		}
		sess, ok := sessions[m.sessionID]
		if !ok {
			continue
		}
		t := Turn{
			SessionID:        m.sessionID,
			MessageID:        m.id,
			ParentMessageID:  m.parentID,
			ParentSessionID:  sess.parentID,
			Agent:            firstNonEmpty(m.mode, m.agent, sess.agent, "squad"),
			Provider:         m.providerID,
			Model:            m.modelID,
			AssistantStart:   m.created,
			AssistantEnd:     m.completed,
			InputTokens:      m.input,
			OutputTokens:     m.output,
			ReasoningTokens:  m.reasoning,
			CacheReadTokens:  m.cacheRead,
			CacheWriteTokens: m.cacheWrite,
			Cost:             m.cost,
			Err:              m.hasErr,
		}
		if t.Model == "" {
			t.Model = sess.modelID
		}
		if t.Provider == "" {
			t.Provider = sess.providerID
		}
		if u, ok := msgs[m.parentID]; ok {
			t.UserStart = u.created
			t.Prompt = strings.TrimSpace(parts.text[u.id])
		} else {
			t.UserStart = m.created
		}
		t.Completion = strings.TrimSpace(parts.text[m.id])
		if t.InputTokens == 0 && t.OutputTokens == 0 {
			if sf, ok := parts.stepFinish[m.id]; ok {
				t.InputTokens = sf.input
				t.OutputTokens = sf.output
				t.ReasoningTokens = sf.reasoning
				t.CacheReadTokens = sf.cacheRead
				t.CacheWriteTokens = sf.cacheWrite
				if t.Cost == 0 {
					t.Cost = sf.cost
				}
			}
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].AssistantStart.Equal(out[j].AssistantStart) {
			return out[i].AssistantStart.Before(out[j].AssistantStart)
		}
		return out[i].MessageID < out[j].MessageID
	})
	return out, nil
}

type sessionRow struct {
	parentID, agent, modelID, providerID string
}

type msgRow struct {
	id, sessionID, parentID, role, mode, agent, modelID, providerID string
	created, completed                                              time.Time
	input, output, reasoning, cacheRead, cacheWrite                 int
	cost                                                            float64
	hasErr                                                          bool
}

type partStore struct {
	text       map[string]string
	stepFinish map[string]msgRow
}

func loadSessions(db *sql.DB, projectRoot string) (map[string]sessionRow, error) {
	rows, err := db.Query(qSessions)
	if err != nil {
		if isMissingSchema(err) {
			schemaSkip("session")
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := map[string]sessionRow{}
	for rows.Next() {
		var id string
		var parent, dir, agent, model sql.NullString
		var created, updated, archived sql.NullInt64
		if err := rows.Scan(&id, &parent, &dir, &agent, &model, &created, &updated, &archived); err != nil {
			return nil, err
		}
		if archived.Valid && archived.Int64 != 0 {
			continue
		}
		if !sameDir(dir.String, projectRoot) {
			continue
		}
		mid, pid := parseSessionModel(model.String)
		out[id] = sessionRow{parentID: parent.String, agent: agent.String, modelID: mid, providerID: pid}
	}
	return out, rows.Err()
}

func loadMessages(db *sql.DB) (map[string]msgRow, error) {
	rows, err := db.Query(qMessages)
	if err != nil {
		if isMissingSchema(err) {
			schemaSkip("message")
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := map[string]msgRow{}
	for rows.Next() {
		var id, sid string
		var created, updated int64
		var data string
		if err := rows.Scan(&id, &sid, &created, &updated, &data); err != nil {
			return nil, err
		}
		m := parseMessageData(id, sid, created, data)
		out[id] = m
	}
	return out, rows.Err()
}

func loadParts(db *sql.DB) (partStore, error) {
	ps := partStore{text: map[string]string{}, stepFinish: map[string]msgRow{}}
	rows, err := db.Query(qParts)
	if err != nil {
		if isMissingSchema(err) {
			schemaSkip("part")
			return ps, nil
		}
		return ps, err
	}
	defer rows.Close()
	for rows.Next() {
		var mid, data string
		if err := rows.Scan(&mid, &data); err != nil {
			return ps, err
		}
		var obj map[string]any
		if json.Unmarshal([]byte(data), &obj) != nil {
			continue
		}
		typ, _ := obj["type"].(string)
		if typ == "text" {
			if s, ok := obj["text"].(string); ok && s != "" {
				if ps.text[mid] != "" {
					ps.text[mid] += "\n"
				}
				ps.text[mid] += s
			}
		}
		if typ == "step-finish" {
			ps.stepFinish[mid] = tokensFromMap(obj)
		}
	}
	return ps, rows.Err()
}

func parseMessageData(id, sid string, created int64, data string) msgRow {
	m := msgRow{id: id, sessionID: sid, created: msTime(created)}
	var obj map[string]any
	if json.Unmarshal([]byte(data), &obj) != nil {
		return m
	}
	m.role, _ = obj["role"].(string)
	m.parentID, _ = obj["parentID"].(string)
	m.mode, _ = obj["mode"].(string)
	m.agent, _ = obj["agent"].(string)
	m.modelID, _ = obj["modelID"].(string)
	m.providerID, _ = obj["providerID"].(string)
	if _, ok := obj["error"]; ok && obj["error"] != nil {
		m.hasErr = true
	}
	if c, ok := asFloat(obj["cost"]); ok {
		m.cost = c
	}
	if tok, ok := obj["tokens"].(map[string]any); ok {
		m.input = asInt(tok["input"])
		m.output = asInt(tok["output"])
		m.reasoning = asInt(tok["reasoning"])
		if cache, ok := tok["cache"].(map[string]any); ok {
			m.cacheRead = asInt(cache["read"])
			m.cacheWrite = asInt(cache["write"])
		}
	}
	if tm, ok := obj["time"].(map[string]any); ok {
		if v, ok := asFloat(tm["created"]); ok {
			m.created = msTime(int64(v))
		}
		if v, ok := asFloat(tm["completed"]); ok {
			m.completed = msTime(int64(v))
		}
	}
	return m
}

func parseSessionModel(raw string) (id, provider string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	var obj struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	}
	if json.Unmarshal([]byte(raw), &obj) != nil {
		return "", ""
	}
	return obj.ID, obj.ProviderID
}

func tokensFromMap(obj map[string]any) msgRow {
	var m msgRow
	if c, ok := asFloat(obj["cost"]); ok {
		m.cost = c
	}
	tok, _ := obj["tokens"].(map[string]any)
	if tok == nil {
		return m
	}
	m.input = asInt(tok["input"])
	m.output = asInt(tok["output"])
	m.reasoning = asInt(tok["reasoning"])
	if cache, ok := tok["cache"].(map[string]any); ok {
		m.cacheRead = asInt(cache["read"])
		m.cacheWrite = asInt(cache["write"])
	}
	return m
}

func sameDir(a, b string) bool {
	a = filepath.ToSlash(filepath.Clean(a))
	b = filepath.ToSlash(filepath.Clean(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func msTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func asInt(v any) int {
	f, ok := asFloat(v)
	if !ok {
		return 0
	}
	return int(f)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isMissingSchema(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "no such column")
}

func schemaSkip(what string) {
	schemaSkipOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "opencode db schema skip: %s\n", what)
	})
}
