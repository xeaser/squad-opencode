package share

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// Upstream is a named template source (local path or later a git URL).
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

// AddUpstream records a source.
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

// SyncUpstream copies opencode/ and squad ignore/git files from a local pack into the project.
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
	return copyTreeIfExists(src, projectRoot)
}

func copyTreeIfExists(src, projectRoot string) (int, error) {
	n := 0
	// If src has opencode/ or .opencode/, copy into project .opencode
	for _, sub := range []string{"opencode", ".opencode"} {
		from := filepath.Join(src, sub)
		if st, err := os.Stat(from); err == nil && st.IsDir() {
			c, err := squad.CopyTree(from, filepath.Join(projectRoot, ".opencode"))
			if err != nil {
				return n, err
			}
			n += c
		}
	}
	return n, nil
}

// InstallPack copies a directory of agents/skills/commands into .opencode.
func InstallPack(projectRoot, packDir string) (int, error) {
	return copyTreeIfExists(packDir, projectRoot)
}

// Link records a shared team directory.
func Link(projectRoot, teamPath string) error {
	if !squad.IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	st, err := os.Stat(teamPath)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("team path must be an existing directory")
	}
	det := squad.Detect(projectRoot)
	cfg := squad.Config{Version: 1, Host: "opencode", Preset: "default"}
	if det.Config != nil {
		cfg = *det.Config
	}
	abs, _ := filepath.Abs(teamPath)
	cfg.LinkPath = abs
	return squad.SaveConfig(projectRoot, cfg)
}
