package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNapArchivesLargeDecisions(t *testing.T) {
	root := primed(t)
	big := strings.Repeat("decision line\n", 400)
	if err := os.WriteFile(filepath.Join(root, ".squad", "decisions.md"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Nap(root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ArchivedDecisions {
		t.Fatal("expected archive")
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "decisions-archive.md")); err != nil {
		t.Fatal(err)
	}
	if len(res.TrimmedKnowledge) == 0 {
		t.Fatal("expected stub knowledge trim")
	}
}

func TestScrubEmails(t *testing.T) {
	root := primed(t)
	p := filepath.Join(root, ".squad", "charter.md")
	if err := os.WriteFile(p, []byte("contact me@example.com please"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ScrubEmails(root, "", false)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "me@example.com") {
		t.Fatal(string(got))
	}
	if !strings.Contains(string(got), "[redacted-email]") {
		t.Fatal(string(got))
	}
}
