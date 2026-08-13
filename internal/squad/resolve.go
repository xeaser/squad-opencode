package squad

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ResolveDir returns the directory that holds live team state.
// Follows ExternalPath when set; LinkPath is informational for status.
func ResolveDir(projectRoot string) string {
	local := SquadDir(projectRoot)
	cfg := readConfigFile(ConfigPath(projectRoot))
	if cfg != nil && cfg.ExternalPath != "" {
		return cfg.ExternalPath
	}
	return local
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
