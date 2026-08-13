package squad

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NapResult is the outcome of context hygiene.
type NapResult struct {
	ArchivedDecisions bool
	TrimmedKnowledge  []string
	Message           string
}

// Nap archives bulky decisions and optionally trims empty knowledge fluff.
func Nap(projectRoot string, dry, deep bool) (NapResult, error) {
	if !IsInitialized(projectRoot) {
		return NapResult{}, fmt.Errorf("not initialized")
	}
	dir := ResolveDir(projectRoot)
	var res NapResult
	decPath := filepath.Join(dir, "decisions.md")
	data, err := os.ReadFile(decPath)
	if err != nil && !os.IsNotExist(err) {
		return res, err
	}
	if err == nil && len(data) > 4000 {
		arch := filepath.Join(dir, "decisions-archive.md")
		stamp := time.Now().UTC().Format("2006-01-02")
		block := fmt.Sprintf("\n\n## Archived %s\n\n%s", stamp, string(data))
		if !dry {
			f, err := os.OpenFile(arch, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return res, err
			}
			_, err = f.WriteString(block)
			f.Close()
			if err != nil {
				return res, err
			}
			stub := "# Decisions\n\n_(older entries moved to decisions-archive.md)_\n"
			if err := os.WriteFile(decPath, []byte(stub), 0o644); err != nil {
				return res, err
			}
		}
		res.ArchivedDecisions = true
	}

	if deep {
		agents := filepath.Join(dir, "agents")
		_ = filepath.WalkDir(agents, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if d.Name() != "knowledge.md" {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			s := strings.TrimSpace(string(b))
			if s == "" || strings.Contains(s, "What I've learned about this project") && len(s) < 120 {
				rel, _ := filepath.Rel(dir, path)
				res.TrimmedKnowledge = append(res.TrimmedKnowledge, filepath.ToSlash(rel))
				if !dry {
					stub := "# Knowledge\n\n_(trimmed by squad-oc nap --deep)_\n"
					_ = os.WriteFile(path, []byte(stub), 0o644)
				}
			}
			return nil
		})
	}

	res.Message = fmt.Sprintf("nap: archived=%v trimmed=%d dry=%v", res.ArchivedDecisions, len(res.TrimmedKnowledge), dry)
	return res, nil
}
