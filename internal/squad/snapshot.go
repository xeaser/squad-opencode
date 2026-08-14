package squad

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Snapshot is a portable .squad export.
type Snapshot struct {
	Version   int               `json:"version"`
	Config    Config            `json:"config"`
	Files     map[string]string `json:"files"`
	HostFiles map[string]string `json:"hostFiles,omitempty"` // paths relative to project root, e.g. ".opencode/agents/lead.md"
}

func shouldSkipExport(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "comms/") && rel != "comms/.gitkeep" {
		return true
	}
	return false
}

func shouldSkipHost(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "node_modules" || strings.HasPrefix(rel, "node_modules/") || strings.Contains(rel, "/node_modules/") {
		return true
	}
	base := path.Base(rel)
	switch base {
	case "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb", "npm-shrinkwrap.json":
		return true
	}
	return false
}

func isHostRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, ".opencode/agents/"),
		strings.HasPrefix(rel, ".opencode/skills/"),
		strings.HasPrefix(rel, ".opencode/commands/"),
		rel == ".opencode/.gitignore",
		rel == "opencode.json":
		return true
	}
	return false
}

func safeProjectRel(rel string) (string, bool) {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." || strings.HasPrefix(rel, "/") || filepath.IsAbs(filepath.FromSlash(rel)) {
		return "", false
	}
	clean := path.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return rel, true
}

func collectHostFiles(projectRoot string) (map[string]string, error) {
	out := map[string]string{}
	add := func(abs, rel string) error {
		rel, ok := safeProjectRel(rel)
		if !ok || shouldSkipHost(rel) || !isHostRel(rel) {
			return nil
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	}

	oc := filepath.Join(projectRoot, ".opencode")
	err := filepath.WalkDir(oc, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(projectRoot, p)
		if err != nil {
			return err
		}
		return add(p, filepath.ToSlash(rel))
	})
	if err != nil {
		return nil, err
	}

	rootJSON := filepath.Join(projectRoot, "opencode.json")
	if _, err := os.Stat(rootJSON); err == nil {
		if err := add(rootJSON, "opencode.json"); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}

// Export writes a JSON snapshot of live team state to destPath.
// Host files under .opencode (and opencode.json) are always recorded.
func Export(projectRoot, destPath string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	dir := ResolveDir(projectRoot)
	cfg := Detect(projectRoot)
	snap := Snapshot{
		Version:   1,
		Files:     map[string]string{},
		HostFiles: map[string]string{},
	}
	if cfg.Config != nil {
		c := *cfg.Config
		c.ExternalPath = ""
		snap.Config = c
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if shouldSkipExport(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snap.Files[rel] = string(data)
		return nil
	})
	if err != nil {
		return err
	}
	host, err := collectHostFiles(projectRoot)
	if err != nil {
		return err
	}
	snap.HostFiles = host
	out, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, out, 0o644)
}

// Import restores a snapshot into projectRoot .squad/ (local, not external).
// When withHost is true, also writes HostFiles. Never overwrites an existing opencode.json.
func Import(projectRoot, srcPath string, withHost bool) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	dir := SquadDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for rel, content := range snap.Files {
		if shouldSkipExport(rel) {
			continue
		}
		dest := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if withHost {
		if err := writeHostFiles(projectRoot, snap.HostFiles); err != nil {
			return err
		}
	}
	cfg := snap.Config
	if cfg.Host == "" {
		cfg.Host = "opencode"
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	cfg.ExternalPath = ""
	return SaveConfig(projectRoot, cfg)
}

func writeHostFiles(projectRoot string, files map[string]string) error {
	for rel, content := range files {
		rel, ok := safeProjectRel(rel)
		if !ok || shouldSkipHost(rel) {
			continue
		}
		if rel == "opencode.json" {
			if _, err := os.Stat(filepath.Join(projectRoot, "opencode.json")); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		dest := filepath.Join(projectRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
