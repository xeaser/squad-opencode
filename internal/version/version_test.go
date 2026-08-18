package version

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionDefaultIsDev(t *testing.T) {
	if Version != "dev" {
		t.Fatalf("Version = %q, want %q", Version, "dev")
	}
}

func TestIdentityMatchesSources(t *testing.T) {
	mod := modulePath(t)
	if Module != mod {
		t.Fatalf("Module = %q, want go.mod %q", Module, mod)
	}
	wantRepo := strings.TrimPrefix(mod, "github.com/")
	if Repo != wantRepo {
		t.Fatalf("Repo = %q, want %q (from Module)", Repo, wantRepo)
	}
	name := goreleaserProjectName(t)
	if Name != name {
		t.Fatalf("Name = %q, want goreleaser project_name %q", Name, name)
	}
}

func TestGoreleaserReleaseIdentity(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	wantX := "-X " + Module + "/internal/version.Version={{.Version}}"
	if !strings.Contains(body, wantX) {
		t.Fatalf("goreleaser ldflags missing %q", wantX)
	}
	if !strings.Contains(body, "GITHUB_REPOSITORY") || !strings.Contains(body, "/internal/version.Repo=") {
		t.Fatal("goreleaser should -X Repo from GITHUB_REPOSITORY when set")
	}
	if strings.Contains(body, "owner: xeaser") || strings.Contains(body, "name: squad-opencode") {
		t.Fatal("goreleaser must not hardcode GitHub owner/name; use GITHUB_REPOSITORY")
	}
}

func TestVersionInjectedByLdflags(t *testing.T) {
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "squad-oc")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	ld := fmt.Sprintf("-X %s/internal/version.Version=9.9.9", Module)
	cmd := exec.Command("go", "build", "-o", bin, "-ldflags", ld, "./cmd/squad-oc")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	got, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("version: %v\n%s", err, got)
	}
	if strings.TrimSpace(string(got)) != "9.9.9" {
		t.Fatalf("squad-oc version = %q, want 9.9.9", got)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// go test runs with cwd = this package directory.
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

func modulePath(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return rest
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Fatal("go.mod has no module line")
	return ""
}

func goreleaserProjectName(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "project_name:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal(".goreleaser.yaml has no project_name")
	return ""
}
