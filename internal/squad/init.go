package squad

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteDefaultPreset scaffolds .squad/ and .opencode/ into projectRoot.
// Idempotent: if already initialized and Force is false, no files are written.
// When opts.Global is set, projectRoot is GlobalSquadDir() (created if missing).
func WriteDefaultPreset(opts InitOptions) (InitResult, error) {
	if opts.Global {
		root, err := GlobalSquadDir()
		if err != nil {
			return InitResult{}, err
		}
		if err := os.MkdirAll(root, 0o755); err != nil {
			return InitResult{}, err
		}
		opts.ProjectRoot = root
	}
	root := opts.ProjectRoot
	if root == "" {
		return InitResult{}, fmt.Errorf("project root is required")
	}
	preset := opts.Preset
	if preset == "" {
		preset = "default"
	}

	if IsInitialized(root) && !opts.Force {
		return InitResult{
			AlreadyInitialized: true,
			ProjectRoot:        root,
			Message:            "Already initialized (.squad/config.json exists). No files changed.",
		}, nil
	}

	var written []string
	tpl := TemplateFS()

	// .squad/ from templates/squad
	if err := copyFS(tpl, "squad", SquadDir(root), root, &written); err != nil {
		return InitResult{}, fmt.Errorf("copy squad templates: %w", err)
	}

	// Apply description placeholders
	for _, rel := range []string{"team.md", "charter.md"} {
		p := filepath.Join(SquadDir(root), rel)
		if err := applyDescriptionFile(p, opts.ProjectDescription); err != nil && !os.IsNotExist(err) {
			return InitResult{}, err
		}
	}

	// .opencode/ from templates/opencode
	if err := copyFS(tpl, "opencode", filepath.Join(root, ".opencode"), root, &written); err != nil {
		return InitResult{}, fmt.Errorf("copy opencode templates: %w", err)
	}

	theme := strings.TrimSpace(opts.Theme)
	var norm string
	if theme != "" {
		var err error
		norm, err = NormalizeTheme(theme)
		if err != nil {
			return InitResult{}, err
		}
	}
	officeBirth := norm == ThemeOffice
	if officeBirth {
		if err := mintOfficeNativeIDs(root); err != nil {
			return InitResult{}, err
		}
	}

	// config.json
	cfg := Config{
		Version:            1,
		Host:               "opencode",
		Preset:             preset,
		ProjectDescription: opts.ProjectDescription,
	}
	if officeBirth {
		cfg.Theme = ThemeOffice
		cfg.ThemeOrigin = ThemeOriginInit
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return InitResult{}, err
	}
	cfgBytes = append(cfgBytes, '\n')
	cfgPath := ConfigPath(root)
	if err := writeTracked(root, cfgPath, cfgBytes, &written); err != nil {
		return InitResult{}, err
	}

	// opencode.json only if missing
	ocJSON := filepath.Join(root, "opencode.json")
	if _, err := os.Stat(ocJSON); os.IsNotExist(err) {
		snippet, err := fs.ReadFile(tpl, "opencode.json")
		if err != nil {
			snippet = []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n")
		}
		if err := writeTracked(root, ocJSON, snippet, &written); err != nil {
			return InitResult{}, err
		}
	}

	// comms dir
	comms := filepath.Join(SquadDir(root), "comms")
	if err := os.MkdirAll(comms, 0o755); err != nil {
		return InitResult{}, err
	}
	gitkeep := filepath.Join(comms, ".gitkeep")
	if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
		if err := writeTracked(root, gitkeep, []byte{}, &written); err != nil {
			return InitResult{}, err
		}
	}

	if officeBirth {
		if _, err := Recast(root); err != nil {
			return InitResult{}, err
		}
	}

	// unique + sort
	written = uniqueSorted(written)

	return InitResult{
		AlreadyInitialized: false,
		ProjectRoot:        root,
		FilesWritten:       written,
		Message:            fmt.Sprintf("Initialized Squad for OpenCode (preset: %s).", preset),
	}, nil
}

// mintOfficeNativeIDs rewrites a default scaffold to Office character IDs.
// Memory ids become michael/jim/dwight/pam. Coordinator stays Squad.
func mintOfficeNativeIDs(root string) error {
	teamFile := filepath.Join(ResolveDir(root), "team.md")
	raw, err := os.ReadFile(teamFile)
	if err != nil {
		return err
	}
	if err := os.WriteFile(teamFile, []byte(rewriteTeamForOfficeBirth(string(raw))), 0o644); err != nil {
		return err
	}
	if err := applyThemeToCharters(root, officeTheme); err != nil {
		return err
	}
	if err := rewriteOfficeCharterPaths(root); err != nil {
		return err
	}
	return renameOfficeAgentDirs(root)
}

func rewriteTeamForOfficeBirth(content string) string {
	content = applyThemeToTeamMarkdown(content, officeTheme)
	for id, name := range officeTheme {
		native := memberID(name)
		content = strings.ReplaceAll(content, ".squad/agents/"+id+"/", ".squad/agents/"+native+"/")
		content = strings.ReplaceAll(content, "@"+id, "@"+native)
	}
	return content
}

func rewriteOfficeCharterPaths(root string) error {
	base := filepath.Join(ResolveDir(root), "agents")
	for id, name := range officeTheme {
		native := memberID(name)
		path := filepath.Join(base, id, "charter.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		next := strings.ReplaceAll(string(raw), ".squad/agents/"+id+"/", ".squad/agents/"+native+"/")
		if next == string(raw) {
			continue
		}
		if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func renameOfficeAgentDirs(root string) error {
	base := filepath.Join(ResolveDir(root), "agents")
	for id, name := range officeTheme {
		native := memberID(name)
		if native == "" || native == id {
			continue
		}
		src := filepath.Join(base, id)
		dst := filepath.Join(base, native)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if _, err := os.Stat(dst); err == nil {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func applyDescriptionFile(path, description string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	repl := "_(describe your project)_"
	if strings.TrimSpace(description) != "" {
		repl = strings.TrimSpace(description)
	}
	out := strings.ReplaceAll(string(data), "{{PROJECT_DESCRIPTION}}", repl)
	return os.WriteFile(path, []byte(out), 0o644)
}

func copyFS(src fs.FS, srcRoot, destRoot, projectRoot string, written *[]string) error {
	return fs.WalkDir(src, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(destRoot, 0o755)
		}
		// embed FS uses slash paths
		rel = filepath.FromSlash(rel)
		target := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		f, err := src.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		return writeTracked(projectRoot, target, data, written)
	})
}

func writeTracked(projectRoot, abs string, data []byte, written *[]string) error {
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return err
	}
	rel, err := filepath.Rel(projectRoot, abs)
	if err != nil {
		rel = abs
	}
	*written = append(*written, filepath.ToSlash(rel))
	return nil
}

func uniqueSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
