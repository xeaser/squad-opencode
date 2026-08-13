package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAndUnknown(t *testing.T) {
	if Execute(nil) != 0 {
		t.Fatal("help")
	}
	if Execute([]string{"nope"}) != 2 {
		t.Fatal("unknown")
	}
	if Execute([]string{"version"}) != 0 {
		t.Fatal("version")
	}
}

func TestInitExportImportViaCLI(t *testing.T) {
	root := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if code := Execute([]string{"init", "--preset", "default", "--description", "cli"}); code != 0 {
		t.Fatal("init")
	}
	if code := Execute([]string{"status"}); code != 0 {
		t.Fatal("status")
	}
	if code := Execute([]string{"upgrade", "--dry-run"}); code != 0 {
		t.Fatal("upgrade")
	}
	snap := filepath.Join(root, "out.json")
	if code := Execute([]string{"export", snap}); code != 0 {
		t.Fatal("export")
	}
	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	if code := Execute([]string{"import", snap}); code != 0 {
		t.Fatal("import")
	}
	b, err := os.ReadFile(filepath.Join(other, ".squad", "team.md"))
	if err != nil || !strings.Contains(string(b), "cli") {
		t.Fatalf("%v %s", err, b)
	}
}

func TestRunRequiresPrompt(t *testing.T) {
	if Execute([]string{"run"}) != 2 {
		t.Fatal()
	}
}
