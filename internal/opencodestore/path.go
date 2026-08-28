// Package opencodestore reads OpenCode SQLite read-only for traces ingest.
package opencodestore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// userHomeDir is os.UserHomeDir; tests replace it.
var userHomeDir = os.UserHomeDir

// ResolveDBPath returns the OpenCode SQLite file.
// explicit is true when OPENCODE_DB or config opencode_db is set.
func ResolveDBPath(projectRoot string, cfg *squad.Config, getenv func(string) string) (path string, explicit bool, err error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if v := strings.TrimSpace(getenv("OPENCODE_DB")); v != "" {
		if v == ":memory:" {
			return "", true, fmt.Errorf("OPENCODE_DB :memory: is not supported")
		}
		if filepath.IsAbs(v) {
			return filepath.Clean(v), true, nil
		}
		dir, err := dataDir(getenv)
		if err != nil {
			return "", true, err
		}
		return filepath.Join(dir, v), true, nil
	}
	if cfg != nil {
		if v := strings.TrimSpace(cfg.OpenCodeDB); v != "" {
			if !filepath.IsAbs(v) {
				if projectRoot == "" {
					return "", true, fmt.Errorf("opencode_db relative path requires project root")
				}
				v = filepath.Join(projectRoot, v)
			}
			return filepath.Clean(v), true, nil
		}
	}
	dir, err := dataDir(getenv)
	if err != nil {
		return "", false, err
	}
	return filepath.Join(dir, "opencode.db"), false, nil
}

func dataDir(getenv func(string) string) (string, error) {
	if v := strings.TrimSpace(getenv("XDG_DATA_HOME")); v != "" {
		return filepath.Join(v, "opencode"), nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "opencode"), nil
}
