package squad

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ResolveDir returns the directory that holds live team state.
// LinkPath (shared team) wins over ExternalPath (moved local team).
func ResolveDir(projectRoot string) string {
	local := SquadDir(projectRoot)
	cfg := readConfigFile(ConfigPath(projectRoot))
	if cfg == nil {
		return local
	}
	if cfg.LinkPath != "" {
		return cfg.LinkPath
	}
	if cfg.ExternalPath != "" {
		return cfg.ExternalPath
	}
	return local
}

// ResolveLinkTarget accepts a project root (with .squad/team.md) or a team directory (team.md).
func ResolveLinkTarget(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("team path required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if fileExists(filepath.Join(abs, "team.md")) {
		return abs, nil
	}
	nested := filepath.Join(abs, ".squad")
	if fileExists(filepath.Join(nested, "team.md")) {
		return nested, nil
	}
	return "", fmt.Errorf("no team.md in %s or %s", abs, nested)
}

// SetLink points this project at an existing team directory. Local config.json stays here.
func SetLink(projectRoot, teamDir string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	cfg := Detect(projectRoot).Config
	if cfg != nil && cfg.ExternalPath != "" {
		return fmt.Errorf("already externalized at %s — internalize first", cfg.ExternalPath)
	}
	if _, err := os.Stat(filepath.Join(teamDir, "team.md")); err != nil {
		return fmt.Errorf("team path must contain team.md")
	}
	next := Config{Version: 1, Host: "opencode", Preset: "default"}
	if cfg != nil {
		next = *cfg
	}
	next.LinkPath = absClean(teamDir)
	return SaveConfig(projectRoot, next)
}

// ClearLink drops the shared-team pointer. Local .squad/ becomes live again.
func ClearLink(projectRoot string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	cfg := Detect(projectRoot).Config
	if cfg == nil || cfg.LinkPath == "" {
		return fmt.Errorf("not linked")
	}
	cfg.LinkPath = ""
	return SaveConfig(projectRoot, *cfg)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func readConfigFile(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil {
		return nil
	}
	return &cfg
}

// SaveConfig writes .squad/config.json.
func SaveConfig(projectRoot string, cfg Config) error {
	if err := os.MkdirAll(SquadDir(projectRoot), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(ConfigPath(projectRoot), data, 0o644)
}

func absClean(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
