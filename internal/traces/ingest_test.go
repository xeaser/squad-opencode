package traces

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestIngestDedupAndSecondTurn(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "oc.db")
	writeIngestFixture(t, dbPath)
	cfg := &squad.Config{OpenCodeDB: dbPath}
	env := func(string) string { return "" }

	n, err := Ingest(root, cfg, env, nil)
	if err != nil || n != 2 {
		t.Fatalf("first n=%d err=%v", n, err)
	}
	spans, err := List(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	var chats, sessions int
	wantChatStart := map[string]time.Time{
		"msg_a": time.UnixMilli(1100).UTC(),
		"msg_b": time.UnixMilli(1400).UTC(),
	}
	wantSessionStart := map[string]time.Time{
		"msg_a": time.UnixMilli(1000).UTC(),
		"msg_b": time.UnixMilli(1300).UTC(),
	}
	for _, s := range spans {
		switch s.Name {
		case NameChat:
			chats++
			if s.MessageID == "" || s.SessionID != "ses_here" {
				t.Fatalf("%+v", s)
			}
			if !s.Start.Equal(wantChatStart[s.MessageID]) {
				t.Fatalf("chat %s start=%v want assistant created %v", s.MessageID, s.Start, wantChatStart[s.MessageID])
			}
		case NameSession:
			sessions++
			if s.MessageID == "" || s.SessionID != "ses_here" {
				t.Fatalf("%+v", s)
			}
			if !s.Start.Equal(wantSessionStart[s.MessageID]) {
				t.Fatalf("session %s start=%v want user %v", s.MessageID, s.Start, wantSessionStart[s.MessageID])
			}
		}
	}
	if chats != 2 || sessions != 2 {
		t.Fatalf("chats=%d sessions=%d", chats, sessions)
	}

	n, err = Ingest(root, cfg, env, nil)
	if err != nil || n != 0 {
		t.Fatalf("second n=%d err=%v", n, err)
	}
}

func TestIngestSkipsKnownMessageIDIngestsLater(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "oc.db")
	writeIngestFixture(t, dbPath)
	start := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	if err := Append(root, Span{
		Name: NameChat, SessionID: "ses_here", MessageID: "msg_a",
		Start: start, End: start.Add(time.Second), Status: "OK",
	}); err != nil {
		t.Fatal(err)
	}
	n, err := Ingest(root, &squad.Config{OpenCodeDB: dbPath}, func(string) string { return "" }, nil)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v want 1", n, err)
	}
	var got []string
	spans, _ := List(root, 0)
	for _, s := range spans {
		if s.Name == NameChat {
			got = append(got, s.MessageID)
		}
	}
	if len(got) != 2 || got[0] != "msg_a" || got[1] != "msg_b" {
		t.Fatalf("chats=%v", got)
	}
}

func TestIngestLegacySkipsEarliestCLI(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "oc.db")
	writeIngestFixture(t, dbPath)
	cfg := &squad.Config{OpenCodeDB: dbPath}
	env := func(string) string { return "" }
	start := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	if err := Append(root, Span{
		Name: "squad-oc.run", SessionID: "ses_here",
		Start: start, End: start.Add(time.Second), Status: "OK",
	}); err != nil {
		t.Fatal(err)
	}
	n, err := Ingest(root, cfg, env, nil)
	if err != nil || n != 1 {
		t.Fatalf("legacy n=%d err=%v", n, err)
	}
	assertNoEarliestCLIChat(t, root)

	// After a later turn is in JSONL, hasMsg would be true under the old
	// gate — second poll must still skip msg_a (durable legacy skip).
	n, err = Ingest(root, cfg, env, nil)
	if err != nil || n != 0 {
		t.Fatalf("second n=%d err=%v", n, err)
	}
	assertNoEarliestCLIChat(t, root)
}

func assertNoEarliestCLIChat(t *testing.T, root string) {
	t.Helper()
	spans, err := List(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range spans {
		if s.Name == NameChat && s.MessageID == "msg_a" {
			t.Fatal("earliest CLI turn ingested")
		}
	}
}

func TestIngestDefaultMissingSilent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg"))
	n, err := Ingest(root, nil, os.Getenv, nil)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestIngestConfiguredMissingErrors(t *testing.T) {
	root := t.TempDir()
	_, err := Ingest(root, &squad.Config{OpenCodeDB: filepath.Join(root, "nope.db")}, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIngestPushFailureKeepsJSONL(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "oc.db")
	writeIngestFixture(t, dbPath)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	n, err := Ingest(root, &squad.Config{OpenCodeDB: dbPath}, os.Getenv, func(context.Context, Settings, Span, *Span) error {
		return context.Canceled
	})
	if err == nil || !IsOTLPPushError(err) {
		t.Fatalf("want otlp push error, got %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d want 2 (jsonl kept for both turns)", n)
	}
	spans, _ := List(root, 0)
	if len(spans) == 0 {
		t.Fatal("jsonl should keep appended turn")
	}
}

func TestIngestExportHasSessionIDNoBodies(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "oc.db")
	writeIngestFixture(t, dbPath)
	n, err := Ingest(root, &squad.Config{OpenCodeDB: dbPath}, func(string) string { return "" }, nil)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	spans, err := List(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "out.otlp.json")
	if err := ExportOTLPFile(spans, dest); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{"session.id", "gen_ai.conversation.id", NameSession, NameChat} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
	if strings.Contains(s, `"text":"a"`) || strings.Contains(s, "SECRET") {
		t.Fatal("export leaked bodies")
	}
	if strings.Contains(s, "gen_ai.input.messages") || strings.Contains(s, "gen_ai.output.messages") {
		t.Fatal("export must not include messages")
	}
}

func writeIngestFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	proj := filepath.Dir(path)
	_, err = db.Exec(`
CREATE TABLE session (
  id text, parent_id text, directory text, agent text, model text,
  time_created integer, time_updated integer, time_archived integer
);
CREATE TABLE message (
  id text, session_id text, time_created integer, time_updated integer, data text
);
CREATE TABLE part (
  id text, message_id text, session_id text, time_created integer, time_updated integer, data text
);
INSERT INTO session VALUES ('ses_here', NULL, ?, 'squad', '{"id":"m","providerID":"p"}', 1, 2, NULL);
INSERT INTO message VALUES
 ('u1', 'ses_here', 1000, 1000, '{"role":"user","time":{"created":1000}}'),
 ('msg_a', 'ses_here', 1100, 1200, '{"role":"assistant","parentID":"u1","mode":"squad","modelID":"m","providerID":"p","cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":1100,"completed":1200}}'),
 ('u2', 'ses_here', 1300, 1300, '{"role":"user","time":{"created":1300}}'),
 ('msg_b', 'ses_here', 1400, 1500, '{"role":"assistant","parentID":"u2","mode":"squad","modelID":"m","providerID":"p","cost":0,"tokens":{"input":2,"output":2,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":1400,"completed":1500}}');
INSERT INTO part VALUES
 ('p1', 'u1', 'ses_here', 1000, 1000, '{"type":"text","text":"a"}'),
 ('p2', 'msg_a', 'ses_here', 1100, 1200, '{"type":"text","text":"A"}'),
 ('p3', 'u2', 'ses_here', 1300, 1300, '{"type":"text","text":"b"}'),
 ('p4', 'msg_b', 'ses_here', 1400, 1500, '{"type":"text","text":"B"}');
`, proj)
	if err != nil {
		t.Fatal(err)
	}
}
