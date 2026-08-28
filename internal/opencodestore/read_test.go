package opencodestore

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestQueryConstantsOmitSecrets(t *testing.T) {
	blob := qSessions + qMessages + qParts
	for _, bad := range []string{"account", "credential", "control_account", "session_share"} {
		if strings.Contains(blob, bad) {
			t.Fatalf("query mentions %s", bad)
		}
	}
}

func TestListTurnsFilterAndMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	writeFixture(t, path)
	db, err := OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proj := `D:/proj`
	turns, err := ListTurns(db, proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("turns=%d %+v", len(turns), turns)
	}
	tr := turns[0]
	if tr.MessageID != "msg_done" || tr.SessionID != "ses_here" {
		t.Fatalf("%+v", tr)
	}
	if tr.Provider != "opencode" || tr.Model != "big-pickle" {
		t.Fatalf("model %+v", tr)
	}
	if tr.InputTokens != 10 || tr.OutputTokens != 2 || tr.Cost != 0 {
		t.Fatalf("tokens %+v", tr)
	}
	if tr.Prompt != "hello" || tr.Completion != "ok" {
		t.Fatalf("bodies %+v", tr)
	}
}

// macOS /var is a symlink to /private/var. OpenCode may store one spelling
// while Getwd() returns the other; ingest must still match the project.
func TestSameDirViaSymlink(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "proj")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(real, link); err != nil {
		if runtime.GOOS == "windows" {
			out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, real).CombinedOutput()
			if jerr != nil {
				t.Skipf("symlink: %v; junction: %v %s", err, jerr, out)
			}
		} else {
			resolved, err2 := filepath.EvalSymlinks(root)
			if err2 != nil || filepath.Clean(resolved) == filepath.Clean(root) {
				t.Skipf("symlink: %v", err)
			}
			real, link = root, resolved
		}
	}
	if !sameDir(real, link) {
		t.Fatalf("sameDir(%q, %q) = false", real, link)
	}
	if sameDir(real, filepath.Join(root, "other")) {
		t.Fatal("different dirs matched")
	}
}

func TestSameDirEmpty(t *testing.T) {
	if sameDir("", "/tmp/proj") || sameDir("/tmp/proj", "") {
		t.Fatal("empty matched")
	}
	if !sameDir("", "") {
		t.Fatal("empty != empty")
	}
}

func writeFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
INSERT INTO session VALUES
 ('ses_here', '', 'D:/proj', 'squad', '{"id":"big-pickle","providerID":"opencode"}', 1000, 2000, NULL),
 ('ses_other', '', 'D:/other', 'squad', '', 1000, 2000, NULL),
 ('ses_arch', '', 'D:/proj', 'squad', '', 1000, 2000, 9);
INSERT INTO message VALUES
 ('msg_user', 'ses_here', 1000, 1000, '{"role":"user","time":{"created":1000}}'),
 ('msg_done', 'ses_here', 1100, 1200, '{"role":"assistant","parentID":"msg_user","mode":"squad","modelID":"big-pickle","providerID":"opencode","cost":0,"tokens":{"input":10,"output":2,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":1100,"completed":1200}}'),
 ('msg_inc', 'ses_here', 1300, 1300, '{"role":"assistant","parentID":"msg_user","mode":"squad","time":{"created":1300}}'),
 ('msg_away', 'ses_other', 1100, 1200, '{"role":"assistant","mode":"squad","time":{"created":1100,"completed":1200}}'),
 ('msg_arch', 'ses_arch', 1100, 1200, '{"role":"assistant","mode":"squad","modelID":"big-pickle","providerID":"opencode","cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":1100,"completed":1200}}');
INSERT INTO part VALUES
 ('p1', 'msg_user', 'ses_here', 1000, 1000, '{"type":"text","text":"hello"}'),
 ('p2', 'msg_done', 'ses_here', 1100, 1200, '{"type":"text","text":"ok"}'),
 ('p3', 'msg_arch', 'ses_arch', 1100, 1200, '{"type":"text","text":"archived"}');
`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
