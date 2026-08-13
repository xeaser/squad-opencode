package squad

import (
	"encoding/json"
	"os"
)

// IsInitialized reports whether .squad/config.json exists.
func IsInitialized(projectRoot string) bool {
	_, err := os.Stat(ConfigPath(projectRoot))
	return err == nil
}

// Detect loads config if present.
func Detect(projectRoot string) DetectResult {
	cfgPath := ConfigPath(projectRoot)
	dir := SquadDir(projectRoot)
	res := DetectResult{
		Initialized: false,
		ProjectRoot: projectRoot,
		SquadDir:    dir,
		ConfigPath:  cfgPath,
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return res
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		res.Initialized = true
		return res
	}
	res.Initialized = true
	res.Config = &cfg
	return res
}
