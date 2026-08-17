package share

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// Marketplace is a named plugin catalog: a local directory or a git URL.
type Marketplace struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Plugin is one discoverable skill in a marketplace.
type Plugin struct {
	Name        string
	Source      string
	Description string
	Triggers    string
}

type marketplaceEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Triggers    json.RawMessage `json:"triggers"`
	Path        string          `json:"path"`
	Source      string          `json:"source"`
}

func marketplaceFile(projectRoot string) string {
	return filepath.Join(squad.SquadDir(projectRoot), "marketplaces.json")
}

func loadMarketplaces(projectRoot string) ([]Marketplace, error) {
	data, err := os.ReadFile(marketplaceFile(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Marketplace
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveMarketplaces(projectRoot string, list []Marketplace) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(squad.SquadDir(projectRoot), 0o755); err != nil {
		return err
	}
	return os.WriteFile(marketplaceFile(projectRoot), data, 0o644)
}

// AddMarketplace records a local directory or git URL.
func AddMarketplace(projectRoot, name, path string) error {
	if name == "" || path == "" {
		return fmt.Errorf("name and path required")
	}
	list, err := loadMarketplaces(projectRoot)
	if err != nil {
		return err
	}
	for i, m := range list {
		if m.Name == name {
			list[i].Path = path
			return saveMarketplaces(projectRoot, list)
		}
	}
	list = append(list, Marketplace{Name: name, Path: path})
	return saveMarketplaces(projectRoot, list)
}

// ListMarketplaces returns recorded catalogs.
func ListMarketplaces(projectRoot string) ([]Marketplace, error) {
	return loadMarketplaces(projectRoot)
}

// RemoveMarketplace drops a named catalog.
func RemoveMarketplace(projectRoot, name string) error {
	list, err := loadMarketplaces(projectRoot)
	if err != nil {
		return err
	}
	out := list[:0]
	for _, m := range list {
		if m.Name != name {
			out = append(out, m)
		}
	}
	return saveMarketplaces(projectRoot, out)
}

// BrowsePlugins lists plugins from one marketplace, or all if name is empty.
func BrowsePlugins(projectRoot, name string) ([]Plugin, error) {
	list, err := loadMarketplaces(projectRoot)
	if err != nil {
		return nil, err
	}
	if name != "" {
		m, ok := findMarketplace(list, name)
		if !ok {
			return nil, fmt.Errorf("unknown marketplace %q", name)
		}
		list = []Marketplace{m}
	}
	var out []Plugin
	for _, m := range list {
		plugins, err := discoverPlugins(m)
		if err != nil {
			return nil, err
		}
		out = append(out, plugins...)
	}
	return out, nil
}

// InstallPlugin copies plugins/<name>/ into .opencode/skills/<name>/.
// --from is optional when exactly one marketplace is registered.
func InstallPlugin(projectRoot, plugin, from string) (int, error) {
	plugin = strings.TrimSpace(plugin)
	if plugin == "" || plugin != filepath.Base(plugin) || plugin == "." || plugin == ".." {
		return 0, fmt.Errorf("invalid plugin name %q", plugin)
	}
	list, err := loadMarketplaces(projectRoot)
	if err != nil {
		return 0, err
	}
	var m Marketplace
	switch {
	case from != "":
		var ok bool
		m, ok = findMarketplace(list, from)
		if !ok {
			return 0, fmt.Errorf("unknown marketplace %q", from)
		}
	case len(list) == 1:
		m = list[0]
	case len(list) == 0:
		return 0, fmt.Errorf("no marketplaces — run: squad-oc marketplace add <name> <path>")
	default:
		return 0, fmt.Errorf("specify --from (multiple marketplaces)")
	}
	dir, cleanup, err := resolveSource(m.Path)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	src, err := locatePluginDir(dir, plugin)
	if err != nil {
		return 0, err
	}
	dest := filepath.Join(projectRoot, ".opencode", "skills", plugin)
	return copyPlugin(src, dest)
}

// UnresolvedMarketplaces returns registered names whose local path cannot be resolved.
// Git URLs are treated as resolvable (no clone).
func UnresolvedMarketplaces(projectRoot string) ([]string, error) {
	list, err := loadMarketplaces(projectRoot)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, m := range list {
		if marketplaceResolves(projectRoot, m.Path) {
			continue
		}
		bad = append(bad, m.Name)
	}
	return bad, nil
}

func marketplaceResolves(projectRoot, src string) bool {
	src = strings.TrimSpace(src)
	if src == "" {
		return false
	}
	if looksLikeGit(src) {
		return true
	}
	if isDir(src) {
		return true
	}
	if !filepath.IsAbs(src) && isDir(filepath.Join(projectRoot, src)) {
		return true
	}
	return false
}

func findMarketplace(list []Marketplace, name string) (Marketplace, bool) {
	for _, m := range list {
		if m.Name == name {
			return m, true
		}
	}
	return Marketplace{}, false
}

func discoverPlugins(m Marketplace) ([]Plugin, error) {
	dir, cleanup, err := resolveSource(m.Path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	found := map[string]Plugin{}
	pluginsDir := filepath.Join(dir, "plugins")
	if isDir(pluginsDir) {
		entries, err := os.ReadDir(pluginsDir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skill := filepath.Join(pluginsDir, e.Name(), "SKILL.md")
			if !isFile(skill) {
				continue
			}
			desc, trig := skillMeta(skill)
			found[e.Name()] = Plugin{
				Name:        e.Name(),
				Source:      m.Name,
				Description: desc,
				Triggers:    trig,
			}
		}
	}

	raw, err := os.ReadFile(filepath.Join(dir, "marketplace.json"))
	if err == nil {
		entries, err := parseMarketplaceJSON(raw)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.Name == "" {
				continue
			}
			p := found[e.Name]
			p.Name = e.Name
			p.Source = m.Name
			if e.Description != "" {
				p.Description = e.Description
			}
			if trig := triggersFromJSON(e.Triggers); trig != "" {
				p.Triggers = trig
			}
			found[e.Name] = p
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	out := make([]Plugin, 0, len(found))
	for _, p := range found {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func locatePluginDir(root, name string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "marketplace.json"))
	if err == nil {
		entries, err := parseMarketplaceJSON(raw)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			if e.Name != name {
				continue
			}
			rel := e.Path
			if rel == "" {
				rel = e.Source
			}
			if rel == "" {
				break
			}
			rel = strings.TrimPrefix(rel, "./")
			cand := filepath.Join(root, filepath.FromSlash(rel))
			if pluginDirOK(cand) {
				return cand, nil
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	cand := filepath.Join(root, "plugins", name)
	if pluginDirOK(cand) {
		return cand, nil
	}
	return "", fmt.Errorf("unknown plugin %q", name)
}

func pluginDirOK(dir string) bool {
	return isDir(dir) && isFile(filepath.Join(dir, "SKILL.md"))
}

func copyPlugin(src, dest string) (int, error) {
	n := 0
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dest, 0o755)
		}
		if skipSquad[filepath.Base(path)] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := writeFile(path, target); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func parseMarketplaceJSON(raw []byte) ([]marketplaceEntry, error) {
	var wrap struct {
		Plugins []marketplaceEntry `json:"plugins"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && wrap.Plugins != nil {
		return wrap.Plugins, nil
	}
	var list []marketplaceEntry
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	return nil, fmt.Errorf("invalid marketplace.json")
}

func triggersFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.Join(list, ", ")
	}
	return ""
}

func skillMeta(path string) (description, triggers string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	meta := parseFrontmatter(string(data))
	description = meta["description"]
	triggers = meta["triggers"]
	if triggers == "" {
		triggers = meta["trigger"]
	}
	return description, triggers
}

func parseFrontmatter(raw string) map[string]string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	trim := strings.TrimSpace(raw)
	if !strings.HasPrefix(trim, "---") {
		return map[string]string{}
	}
	rest := strings.TrimPrefix(trim, "---")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	var listKey string
	var listVals []string
	flush := func() {
		if listKey != "" {
			out[listKey] = strings.Join(listVals, ", ")
			listKey = ""
			listVals = nil
		}
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.HasPrefix(s, "- ") && listKey != "" {
			listVals = append(listVals, strings.TrimSpace(strings.TrimPrefix(s, "- ")))
			continue
		}
		flush()
		k, v, ok := strings.Cut(s, ":")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if v == "" {
			listKey = k
			continue
		}
		out[k] = v
	}
	flush()
	return out
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
