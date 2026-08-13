package squad

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

var emailRE = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// ScrubEmails replaces email addresses in markdown under dir (default: resolved .squad).
func ScrubEmails(projectRoot, dir string, dry bool) (int, error) {
	if dir == "" {
		if !IsInitialized(projectRoot) {
			return 0, fmt.Errorf("not initialized")
		}
		dir = ResolveDir(projectRoot)
	}
	changed := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".txt" && ext != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		next := emailRE.ReplaceAll(data, []byte("[redacted-email]"))
		if string(next) == string(data) {
			return nil
		}
		changed++
		if !dry {
			return os.WriteFile(path, next, 0o644)
		}
		return nil
	})
	return changed, err
}
