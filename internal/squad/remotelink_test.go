package squad

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLooksLikeGit(t *testing.T) {
	if !LooksLikeGit("https://github.com/acme/squad-platform") {
		t.Fatal("https")
	}
	if !LooksLikeGit("git@github.com:acme/squad-platform.git") {
		t.Fatal("git@")
	}
	if !LooksLikeGit(filepath.Join(t.TempDir(), "platform.git")) {
		t.Fatal(".git suffix")
	}
	if LooksLikeGit(t.TempDir()) {
		t.Fatal("plain dir is not a git url")
	}
}

func TestEnsureRemoteCheckoutUsesLocalGitRemote(t *testing.T) {
	home := isolateHome(t)
	remote := makeTeamRemote(t, "platform-team")

	teamDir, ref, sha, err := EnsureRemoteCheckout(remote)
	if err != nil {
		t.Fatal(err)
	}
	if ref == "" || sha == "" {
		t.Fatalf("ref=%q sha=%q", ref, sha)
	}
	if _, err := os.Stat(filepath.Join(teamDir, "team.md")); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(teamDir, "team.md"))
	if !strings.Contains(string(body), "platform-team") {
		t.Fatalf("team.md: %s", body)
	}
	cache, err := LinksCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(teamDir, filepath.Join(home, ".squad-oc", "links")) &&
		!strings.HasPrefix(filepath.Dir(teamDir), cache) {
		t.Fatalf("checkout not under links cache: %s", teamDir)
	}
}

func TestEnsureRemoteCheckoutMissingTeam(t *testing.T) {
	isolateHome(t)
	empty := t.TempDir()
	runGit(t, empty, "init")
	runGit(t, empty, "config", "user.email", "test@example.com")
	runGit(t, empty, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(empty, "README.md"), []byte("no team"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, empty, "add", ".")
	runGit(t, empty, "commit", "-m", "empty")
	bare := filepath.Join(t.TempDir(), "empty.git")
	runGit(t, empty, "clone", "--bare", empty, bare)
	if _, _, _, err := EnsureRemoteCheckout(bare); err == nil {
		t.Fatal("expected no team.md error")
	}
}

func TestSyncRemoteCheckoutUpdatesSHA(t *testing.T) {
	isolateHome(t)
	work, remote := makeTeamRemotePair(t, "v1")
	_, ref, sha1, err := EnsureRemoteCheckout(remote)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".squad", "decisions.md"), []byte("# Decisions\n\nupdated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "v2")
	runGit(t, work, "push", remote, "HEAD")

	teamDir, _, sha2, err := SyncRemoteCheckout(remote, ref)
	if err != nil {
		t.Fatal(err)
	}
	if sha2 == "" || sha2 == sha1 {
		t.Fatalf("sha did not move: %s -> %s", sha1, sha2)
	}
	body, _ := os.ReadFile(filepath.Join(teamDir, "decisions.md"))
	if !strings.Contains(string(body), "updated") {
		t.Fatalf("checkout stale: %s", body)
	}
}

func makeTeamRemote(t *testing.T, desc string) string {
	t.Helper()
	_, remote := makeTeamRemotePair(t, desc)
	return remote
}

func makeTeamRemotePair(t *testing.T, desc string) (work, remote string) {
	t.Helper()
	work = t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: work, ProjectDescription: desc}); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "init")
	runGit(t, work, "config", "user.email", "test@example.com")
	runGit(t, work, "config", "user.name", "test")
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "team")
	remote = filepath.Join(t.TempDir(), "platform.git")
	runGit(t, work, "clone", "--bare", work, remote)
	return work, remote
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
