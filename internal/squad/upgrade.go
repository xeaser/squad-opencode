package squad

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// UpgradeOptions controls UpgradeHostFiles.
type UpgradeOptions struct {
	ProjectRoot string
	DryRun      bool
	Force       bool
}

// UpgradeResult summarizes host-file refresh.
type UpgradeResult struct {
	Updated   []string
	Unchanged []string
	Created   []string
	Message   string
}

// ownedTemplateCopies maps embed path (under templates/) → dest path relative to project root.
// Team memory under .squad/ (except .gitignore) is intentionally absent.
func ownedTemplateCopies() [][2]string {
	tpl := TemplateFS()
	var pairs [][2]string

	add := func(embedPath, destRel string) {
		pairs = append(pairs, [2]string{embedPath, destRel})
	}

	_ = fs.WalkDir(tpl, "opencode", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(path, "opencode/")
		if rel == "" || rel == path {
			return nil
		}
		add(path, ".opencode/"+rel)
		return nil
	})

	// Only .squad/.gitignore from the squad template tree
	if _, err := fs.Stat(tpl, "squad/.gitignore"); err == nil {
		add("squad/.gitignore", ".squad/.gitignore")
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i][1] < pairs[j][1] })
	return pairs
}

// UpgradeHostFiles refreshes Squad-owned OpenCode/host files from embedded templates.
// Never writes team state (team.md, charters, knowledge, decisions, config).
func UpgradeHostFiles(opts UpgradeOptions) (UpgradeResult, error) {
	root := opts.ProjectRoot
	if root == "" {
		return UpgradeResult{}, fmt.Errorf("project root is required")
	}
	if !IsInitialized(root) {
		return UpgradeResult{}, fmt.Errorf("not initialized — run: squad-oc init --preset default")
	}

	tpl := TemplateFS()
	var res UpgradeResult

	for _, pair := range ownedTemplateCopies() {
		embedPath, destRel := pair[0], pair[1]
		data, err := fs.ReadFile(tpl, embedPath)
		if err != nil {
			return res, fmt.Errorf("read template %s: %w", embedPath, err)
		}
		dest := filepath.Join(root, filepath.FromSlash(destRel))
		existing, err := os.ReadFile(dest)
		missing := err != nil
		if missing && !os.IsNotExist(err) {
			return res, err
		}

		same := !missing && bytes.Equal(existing, data)
		if same && !opts.Force {
			res.Unchanged = append(res.Unchanged, destRel)
			continue
		}

		if opts.DryRun {
			if missing {
				res.Created = append(res.Created, destRel)
			} else {
				res.Updated = append(res.Updated, destRel)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return res, err
		}
		if missing {
			res.Created = append(res.Created, destRel)
		} else {
			res.Updated = append(res.Updated, destRel)
		}
	}

	res.Message = formatUpgradeMessage(opts.DryRun, res)
	return res, nil
}

func formatUpgradeMessage(dry bool, res UpgradeResult) string {
	prefix := "Upgraded"
	if dry {
		prefix = "Dry run"
	}
	return fmt.Sprintf(
		"%s: %d updated, %d created, %d unchanged. Team state (.squad/team, decisions, knowledge) was not touched.",
		prefix, len(res.Updated), len(res.Created), len(res.Unchanged),
	)
}
