package squad

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Snapshot is a portable .squad export.
type Snapshot struct {
	Version int               `json:"version"`
	Config  Config            `json:"config"`
	Files   map[string]string `json:"files"`
}

func shouldSkipExport(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "comms/") && rel != "comms/.gitkeep" {
		return true
	}
	return false
}

// Export writes a JSON snapshot of live team state to destPath.
func Export(projectRoot, destPath string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	dir := ResolveDir(projectRoot)
	cfg := Detect(projectRoot)
	snap := Snapshot{
		Version: 1,
		Files:   map[string]string{},
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
func Import(projectRoot, srcPath string) error {
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
