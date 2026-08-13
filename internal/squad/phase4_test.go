package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func primed(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root, ProjectDescription: "Snap"}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestExportImportRoundTrip(t *testing.T) {
	src := primed(t)
	know := filepath.Join(src, ".squad", "agents", "lead", "knowledge.md")
	if err := os.WriteFile(know, []byte("secret-knowledge"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "snap.json")
	if err := Export(src, out); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := Import(dest, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, ".squad", "agents", "lead", "knowledge.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret-knowledge" {
		t.Fatalf("got %q", got)
	}
	if !IsInitialized(dest) {
		t.Fatal("import should initialize")
	}
}

func TestExternalizeInternalize(t *testing.T) {
	root := primed(t)
	ext := filepath.Join(t.TempDir(), "ext")
	path, err := ExternalizeTo(root, ext)
	if err != nil {
		t.Fatal(err)
	}
	if path != ext {
		t.Fatal(path)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "team.md")); !os.IsNotExist(err) {
		t.Fatal("local team.md should be gone")
	}
	members, err := ReadTeam(root)
	if err != nil || len(members) < 4 {
		t.Fatalf("resolve team: %v %d", err, len(members))
	}
	if err := Internalize(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".squad", "team.md")); err != nil {
		t.Fatal("team.md should be back")
	}
	if Detect(root).Config.ExternalPath != "" {
		t.Fatal("pointer should clear")
	}
}

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
