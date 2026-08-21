package squad

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LooksLikeGit reports https://, git@, or a *.git path. Same rule as pack/upstream.
func LooksLikeGit(s string) bool {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") || strings.HasPrefix(s, "git@") {
		return true
	}
	return strings.HasSuffix(s, ".git")
}

func normalizeLinkURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimSuffix(u, "/")
	return strings.TrimSuffix(u, ".git")
}

func linkCachePath(url string) (string, error) {
	root, err := LinksCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(strings.ToLower(normalizeLinkURL(url))))
	return filepath.Join(root, hex.EncodeToString(sum[:16])), nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func isGitDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func checkoutRefSHA(dir string) (ref, sha string, err error) {
	ref, err = git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", err
	}
	if ref == "HEAD" {
		ref = ""
	}
	sha, err = git(dir, "rev-parse", "HEAD")
	return ref, sha, err
}

func resolveClonedTeam(url, dest string) (teamDir, ref, sha string, err error) {
	teamDir, err = ResolveLinkTarget(dest)
	if err != nil {
		return "", "", "", fmt.Errorf("cloned %s but %w", url, err)
	}
	ref, sha, err = checkoutRefSHA(dest)
	if err != nil {
		return "", "", "", err
	}
	return teamDir, ref, sha, nil
}

// EnsureRemoteCheckout clones url into the links cache (or reuses it) and returns the team dir.
func EnsureRemoteCheckout(url string) (teamDir, ref, sha string, err error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", "", fmt.Errorf("git url required")
	}
	dest, err := linkCachePath(url)
	if err != nil {
		return "", "", "", err
	}
	if isGitDir(dest) {
		return SyncRemoteCheckout(url, "")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", "", "", err
	}
	_ = os.RemoveAll(dest)
	if _, err := git("", "clone", "--depth", "1", "--", url, dest); err != nil {
		_ = os.RemoveAll(dest)
		return "", "", "", err
	}
	return resolveClonedTeam(url, dest)
}

// SyncRemoteCheckout fetches origin and hard-resets the cached checkout.
func SyncRemoteCheckout(url, ref string) (teamDir, nextRef, sha string, err error) {
	url = strings.TrimSpace(url)
	dest, err := linkCachePath(url)
	if err != nil {
		return "", "", "", err
	}
	if !isGitDir(dest) {
		return EnsureRemoteCheckout(url)
	}
	if strings.TrimSpace(ref) == "" {
		ref, err = git(dest, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return "", "", "", err
		}
	}
	if ref == "" || ref == "HEAD" {
		if _, err := git(dest, "fetch", "--depth", "1", "origin"); err != nil {
			return "", "", "", err
		}
		if _, err := git(dest, "reset", "--hard", "FETCH_HEAD"); err != nil {
			return "", "", "", err
		}
	} else {
		if _, err := git(dest, "fetch", "--depth", "1", "origin", ref); err != nil {
			return "", "", "", err
		}
		if _, err := git(dest, "checkout", ref); err != nil {
			return "", "", "", err
		}
		if _, err := git(dest, "reset", "--hard", "origin/"+ref); err != nil {
			return "", "", "", err
		}
	}
	return resolveClonedTeam(url, dest)
}
