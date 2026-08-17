package share

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// Upstream is a named template source: a local directory or a git URL.
type Upstream struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func upstreamFile(projectRoot string) string {
	return filepath.Join(squad.SquadDir(projectRoot), "upstreams.json")
}

func loadUpstreams(projectRoot string) ([]Upstream, error) {
	data, err := os.ReadFile(upstreamFile(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Upstream
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveUpstreams(projectRoot string, list []Upstream) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(squad.SquadDir(projectRoot), 0o755); err != nil {
		return err
	}
	return os.WriteFile(upstreamFile(projectRoot), data, 0o644)
}

// AddUpstream records a local directory or git URL.
func AddUpstream(projectRoot, name, path string) error {
	if name == "" || path == "" {
		return fmt.Errorf("name and path required")
	}
	list, err := loadUpstreams(projectRoot)
	if err != nil {
		return err
	}
	for i, u := range list {
		if u.Name == name {
			list[i].Path = path
			return saveUpstreams(projectRoot, list)
		}
	}
	list = append(list, Upstream{Name: name, Path: path})
	return saveUpstreams(projectRoot, list)
}

// ListUpstreams returns recorded sources.
func ListUpstreams(projectRoot string) ([]Upstream, error) {
	return loadUpstreams(projectRoot)
}

// RemoveUpstream drops a named source.
func RemoveUpstream(projectRoot, name string) error {
	list, err := loadUpstreams(projectRoot)
	if err != nil {
		return err
	}
	out := list[:0]
	for _, u := range list {
		if u.Name != name {
			out = append(out, u)
		}
	}
	return saveUpstreams(projectRoot, out)
}

// SyncUpstream copies host files (and new .squad snippets) from a named source.
func SyncUpstream(projectRoot, name string) (int, error) {
	list, err := loadUpstreams(projectRoot)
	if err != nil {
		return 0, err
	}
	var src string
	for _, u := range list {
		if u.Name == name {
			src = u.Path
			break
		}
	}
	if src == "" {
		return 0, fmt.Errorf("unknown upstream %q", name)
	}
	return ApplySource(projectRoot, src)
}

// InstallPack copies one source into the project (no registration).
func InstallPack(projectRoot, packDir string) (int, error) {
	return ApplySource(projectRoot, packDir)
}

// CloneGit clones url into dest. Tests may replace this.
var CloneGit = func(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", "--", url, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w\n%s", url, err, out)
	}
	return nil
}

func looksLikeGit(s string) bool {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") || strings.HasPrefix(s, "git@") {
		return true
	}
	return strings.HasSuffix(s, ".git")
}

// ApplySource copies .opencode host files from src, then any new .squad snippets.
// Existing team memory (team.md, decisions, config, knowledge) is never overwritten.
func ApplySource(projectRoot, src string) (int, error) {
	dir, cleanup, err := resolveSource(src)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	return applyLocal(projectRoot, dir)
}

func resolveSource(src string) (dir string, cleanup func(), err error) {
	cleanup = func() {}
	src = strings.TrimSpace(src)
	if src == "" {
		return "", cleanup, fmt.Errorf("source required")
	}
	if st, err := os.Stat(src); err == nil && st.IsDir() {
		abs, err := filepath.Abs(src)
		if err != nil {
			return "", cleanup, err
		}
		return abs, cleanup, nil
	}
	if !looksLikeGit(src) {
		return "", cleanup, fmt.Errorf("source not found: %s", src)
	}
	tmp, err := os.MkdirTemp("", "squad-oc-src-*")
	if err != nil {
		return "", cleanup, err
	}
	dest := filepath.Join(tmp, "repo")
	if err := CloneGit(src, dest); err != nil {
		os.RemoveAll(tmp)
		return "", cleanup, err
	}
	return dest, func() { os.RemoveAll(tmp) }, nil
}

func applyLocal(projectRoot, src string) (int, error) {
	n := 0
	hostDest := filepath.Join(projectRoot, ".opencode")
	copiedHost := false
	for _, sub := range []string{".opencode", "opencode"} {
		from := filepath.Join(src, sub)
		if !isDir(from) {
			continue
		}
		c, err := squad.CopyTree(from, hostDest)
		if err != nil {
			return n, err
		}
		n += c
		copiedHost = true
	}
	if !copiedHost && isHostTree(src) {
		c, err := squad.CopyTree(src, hostDest)
		if err != nil {
			return n, err
		}
		n += c
	}
	for _, sub := range []string{".squad", "squad"} {
		from := filepath.Join(src, sub)
		if !isDir(from) {
			continue
		}
		c, err := copySquadSnippets(from, filepath.Join(projectRoot, ".squad"))
		if err != nil {
			return n, err
		}
		n += c
	}
	c, err := copyPackRootMCP(src, projectRoot)
	if err != nil {
		return n, err
	}
	n += c
	if n == 0 {
		return 0, fmt.Errorf("no .opencode/ or .squad/ content in %s", src)
	}
	return n, nil
}

func copyPackRootMCP(src, projectRoot string) (int, error) {
	from := filepath.Join(src, "mcp-config.json")
	st, err := os.Stat(from)
	if err != nil || st.IsDir() {
		return 0, nil
	}
	dest := filepath.Join(squad.ResolveDir(projectRoot), "mcp-config.json")
	if _, err := os.Stat(dest); err == nil {
		return 0, nil
	}
	if err := writeFile(from, dest); err != nil {
		return 0, err
	}
	return 1, nil
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func isHostTree(dir string) bool {
	for _, name := range []string{"agents", "skills", "commands"} {
		if isDir(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

var skipSquad = map[string]bool{
	"team.md":              true,
	"decisions.md":         true,
	"decisions-archive.md": true,
	"config.json":          true,
	"upstreams.json":       true,
}

func copySquadSnippets(src, dest string) (int, error) {
	n := 0
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if filepath.Base(path) == "comms" {
				return fs.SkipDir
			}
			return nil
		}
		if skipSquad[filepath.Base(path)] {
			return nil
		}
		target := filepath.Join(dest, rel)
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		if err := writeFile(path, target); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func writeFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// Link points this project at a shared team (project root or .squad dir).
func Link(projectRoot, teamPath string) (string, error) {
	dest, err := squad.ResolveLinkTarget(teamPath)
	if err != nil {
		return "", err
	}
	if err := squad.SetLink(projectRoot, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// Unlink clears the shared-team pointer.
func Unlink(projectRoot string) error {
	return squad.ClearLink(projectRoot)
}
