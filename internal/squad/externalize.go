package squad

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultExternalRoot is ~/.squad-oc/projects
func DefaultExternalRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "squad-oc-projects")
	}
	return filepath.Join(home, ".squad-oc", "projects")
}

// Externalize moves live .squad contents to dest and leaves a pointer config.
func Externalize(projectRoot, key string) (string, error) {
	if !IsInitialized(projectRoot) {
		return "", fmt.Errorf("not initialized")
	}
	cfg := Detect(projectRoot).Config
	if cfg != nil && cfg.ExternalPath != "" {
		return "", fmt.Errorf("already externalized at %s", cfg.ExternalPath)
	}
	if key == "" {
		key = filepath.Base(absClean(projectRoot))
	}
	return ExternalizeTo(projectRoot, filepath.Join(DefaultExternalRoot(), key))
}

// ExternalizeTo moves state to dest (used by tests to avoid $HOME).
func ExternalizeTo(projectRoot, dest string) (string, error) {
	src := SquadDir(projectRoot)
	cfg := Detect(projectRoot).Config
	if cfg != nil && cfg.LinkPath != "" {
		return "", fmt.Errorf("already linked to %s — unlink first", cfg.LinkPath)
	}
	if _, err := CopyTree(src, dest); err != nil {
		return "", err
	}
	if err := clearDirExceptConfig(src); err != nil {
		return "", err
	}
	next := Config{Version: 1, Host: "opencode", Preset: "default", ExternalPath: dest}
	if cfg != nil {
		next = *cfg
		next.ExternalPath = dest
	}
	if err := SaveConfig(projectRoot, next); err != nil {
		return "", err
	}
	return dest, nil
}

// Internalize copies external state back into .squad/ and clears the pointer.
func Internalize(projectRoot string) error {
	if !IsInitialized(projectRoot) {
		return fmt.Errorf("not initialized")
	}
	cfg := Detect(projectRoot).Config
	if cfg == nil || cfg.ExternalPath == "" {
		return fmt.Errorf("not externalized")
	}
	src := cfg.ExternalPath
	dest := SquadDir(projectRoot)
	if _, err := CopyTree(src, dest); err != nil {
		return err
	}
	cfg.ExternalPath = ""
	return SaveConfig(projectRoot, *cfg)
}

func clearDirExceptConfig(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == "config.json" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
