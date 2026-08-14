package squad

import (
	"encoding/json"
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
	if err := Import(dest, out, false); err != nil {
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

func TestImportWithHostOptional(t *testing.T) {
	src := primed(t)
	if err := os.WriteFile(filepath.Join(src, ".opencode", "package.json"), []byte(`{"private":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(src, ".opencode", "node_modules", "x")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "index.js"), []byte("skip-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "snap.json")
	if err := Export(src, out); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.HostFiles[".opencode/agents/lead.md"] == "" {
		t.Fatal("export should record .opencode/agents/lead.md")
	}
	if _, ok := snap.HostFiles[".opencode/.gitignore"]; !ok {
		t.Fatal("export should record .opencode/.gitignore")
	}
	if _, ok := snap.HostFiles["opencode.json"]; !ok {
		t.Fatal("export should record opencode.json")
	}
	if _, ok := snap.HostFiles[".opencode/package.json"]; ok {
		t.Fatal("export should skip package.json")
	}
	for rel := range snap.HostFiles {
		if strings.Contains(rel, "node_modules") {
			t.Fatalf("export should skip node_modules: %s", rel)
		}
	}

	dest := t.TempDir()
	if err := Import(dest, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".opencode", "agents", "lead.md")); !os.IsNotExist(err) {
		t.Fatal("default import should not write host files")
	}
	if !IsInitialized(dest) {
		t.Fatal("import should initialize")
	}

	dest2 := t.TempDir()
	if err := Import(dest2, out, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest2, ".opencode", "agents", "lead.md")); err != nil {
		t.Fatal("with-host should write .opencode/agents/lead.md")
	}
	if _, err := os.Stat(filepath.Join(dest2, ".opencode", ".gitignore")); err != nil {
		t.Fatal("with-host should write .opencode/.gitignore")
	}
}

func TestImportWithHostSkipsExistingOpencodeJSON(t *testing.T) {
	src := primed(t)
	out := filepath.Join(t.TempDir(), "snap.json")
	if err := Export(src, out); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	existing := filepath.Join(dest, "opencode.json")
	if err := os.WriteFile(existing, []byte(`{"keep":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Import(dest, out, true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"keep":true}` {
		t.Fatalf("opencode.json overwritten: %s", got)
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
