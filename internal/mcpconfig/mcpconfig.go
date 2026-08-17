package mcpconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/xeaser/squad-opencode/internal/squad"
)

const schemaURL = "https://opencode.ai/config.json"

// Server is the OpenCode-native MCP server form.
type Server struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// ListItem is a secret-free source-vs-applied row.
type ListItem struct {
	Name    string
	Source  string
	Enabled bool
	Applied bool
}

// OrgPath is .squad/mcp-config.json after link resolution.
func OrgPath(projectRoot string) string {
	return filepath.Join(squad.ResolveDir(projectRoot), "mcp-config.json")
}

// PackPath is optional pack-root mcp-config.json.
func PackPath(projectRoot string) string {
	return filepath.Join(projectRoot, "mcp-config.json")
}

// Parse decodes Copilot mcpServers or OpenCode mcp JSON (JSONC allowed).
func Parse(data []byte) (map[string]Server, error) {
	clean, err := stripJSONC(data)
	if err != nil {
		return nil, err
	}
	var raw struct {
		MCP        map[string]json.RawMessage `json:"mcp"`
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(clean, &raw); err != nil {
		return nil, fmt.Errorf("parse mcp-config: %w", err)
	}
	out := map[string]Server{}
	for name, body := range raw.MCP {
		s, err := parseServer(body)
		if err != nil {
			return nil, fmt.Errorf("mcp %q: %w", name, err)
		}
		out[name] = s
	}
	for name, body := range raw.MCPServers {
		s, err := parseServer(body)
		if err != nil {
			return nil, fmt.Errorf("mcpServers %q: %w", name, err)
		}
		out[name] = s
	}
	return out, nil
}

// ParseFile reads and parses path.
func ParseFile(path string) (map[string]Server, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

type rawServer struct {
	Type        string            `json:"type"`
	Command     json.RawMessage   `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Environment map[string]string `json:"environment"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Enabled     *bool             `json:"enabled"`
}

func parseServer(body json.RawMessage) (Server, error) {
	var raw rawServer
	if err := json.Unmarshal(body, &raw); err != nil {
		return Server{}, err
	}
	s := Server{
		Type:    strings.TrimSpace(raw.Type),
		URL:     strings.TrimSpace(raw.URL),
		Headers: rewriteMap(raw.Headers),
		Enabled: true,
	}
	if raw.Enabled != nil {
		s.Enabled = *raw.Enabled
	}
	cmd, err := parseCommand(raw.Command, raw.Args)
	if err != nil {
		return Server{}, err
	}
	s.Command = cmd
	env := map[string]string{}
	for k, v := range raw.Environment {
		env[k] = rewriteEnv(v)
	}
	for k, v := range raw.Env {
		env[k] = rewriteEnv(v)
	}
	if len(env) > 0 {
		s.Environment = env
	}
	if s.Type == "" {
		if s.URL != "" {
			s.Type = "remote"
		} else {
			s.Type = "local"
		}
	}
	if s.Type == "local" && len(s.Command) == 0 {
		return Server{}, fmt.Errorf("local server needs command")
	}
	if s.Type == "remote" && s.URL == "" {
		return Server{}, fmt.Errorf("remote server needs url")
	}
	return s, nil
}

func parseCommand(raw json.RawMessage, args []string) ([]string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	trim := bytes.TrimSpace(raw)
	if trim[0] == '[' {
		var parts []string
		if err := json.Unmarshal(trim, &parts); err != nil {
			return nil, fmt.Errorf("command: %w", err)
		}
		return parts, nil
	}
	var s string
	if err := json.Unmarshal(trim, &s); err != nil {
		return nil, fmt.Errorf("command: %w", err)
	}
	out := []string{s}
	out = append(out, args...)
	return out, nil
}

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func rewriteEnv(s string) string {
	return envVarPattern.ReplaceAllString(s, `{env:$1}`)
}

func rewriteMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = rewriteEnv(v)
	}
	return out
}

// Validate rejects hardcoded token-like values.
func Validate(servers map[string]Server) error {
	for name, s := range servers {
		for k, v := range s.Environment {
			if tok := tokenLike(v); tok != "" {
				return fmt.Errorf("hardcoded secret-like value %s in MCP server %q environment %s; use ${VAR} or {env:VAR}", tok, name, k)
			}
		}
		for k, v := range s.Headers {
			if tok := tokenLike(v); tok != "" {
				return fmt.Errorf("hardcoded secret-like value %s in MCP server %q header %s; use ${VAR} or {env:VAR}", tok, name, k)
			}
		}
		if tok := tokenLike(s.URL); tok != "" {
			return fmt.Errorf("hardcoded secret-like value %s in MCP server %q url; use ${VAR} or {env:VAR}", tok, name)
		}
		for _, part := range s.Command {
			if tok := tokenLike(part); tok != "" {
				return fmt.Errorf("hardcoded secret-like value %s in MCP server %q command; use ${VAR} or {env:VAR}", tok, name)
			}
		}
	}
	return nil
}

func tokenLike(s string) string {
	if s == "" || isEnvRef(s) {
		return ""
	}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '"' || r == '\''
	}) {
		if strings.HasPrefix(part, "sk-") {
			return "sk-"
		}
		if strings.HasPrefix(part, "ghp_") {
			return "ghp_"
		}
	}
	return ""
}

func isEnvRef(s string) bool {
	t := strings.TrimSpace(s)
	if envVarPattern.MatchString(t) {
		return true
	}
	return strings.Contains(t, "{env:")
}

// Merge writes servers into an existing opencode.json document.
// Org servers overwrite same-name entries; $schema and unrelated keys stay.
func Merge(opencodeJSON []byte, servers map[string]Server) ([]byte, error) {
	if err := Validate(servers); err != nil {
		return nil, err
	}
	doc := map[string]any{}
	if len(bytes.TrimSpace(opencodeJSON)) > 0 {
		if err := json.Unmarshal(opencodeJSON, &doc); err != nil {
			return nil, fmt.Errorf("parse opencode.json: %w", err)
		}
	}
	if _, ok := doc["$schema"]; !ok {
		doc["$schema"] = schemaURL
	}
	mcp, _ := doc["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
	}
	for name, s := range servers {
		mcp[name] = serverObject(s)
	}
	doc["mcp"] = mcp
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}

func serverObject(s Server) map[string]any {
	obj := map[string]any{
		"type":    s.Type,
		"enabled": s.Enabled,
	}
	if s.Type == "remote" {
		obj["url"] = s.URL
		if len(s.Headers) > 0 {
			obj["headers"] = s.Headers
		}
		return obj
	}
	if len(s.Command) > 0 {
		obj["command"] = s.Command
	}
	if len(s.Environment) > 0 {
		obj["environment"] = s.Environment
	}
	return obj
}

// LoadSources reads pack-root then org file. Org wins on same-name conflict.
func LoadSources(projectRoot string) (map[string]Server, map[string]string, error) {
	out := map[string]Server{}
	src := map[string]string{}
	pack := PackPath(projectRoot)
	org := OrgPath(projectRoot)
	same := sameFile(pack, org)
	if st, err := os.Stat(pack); err == nil && !st.IsDir() {
		got, err := ParseFile(pack)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", pack, err)
		}
		for name, s := range got {
			out[name] = s
			src[name] = "pack"
		}
	}
	if !same {
		if st, err := os.Stat(org); err == nil && !st.IsDir() {
			got, err := ParseFile(org)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", org, err)
			}
			for name, s := range got {
				out[name] = s
				src[name] = "org"
			}
		}
	}
	return out, src, nil
}

func sameFile(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err != nil {
		return a == b
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return a == b
	}
	return aa == bb
}

// Apply merges org + pack MCP sources into projectRoot/opencode.json.
func Apply(projectRoot string) error {
	servers, _, err := LoadSources(projectRoot)
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		org, pack := OrgPath(projectRoot), PackPath(projectRoot)
		if !fileExists(org) && !fileExists(pack) {
			return fmt.Errorf("no mcp-config.json at %s or pack root %s", org, pack)
		}
	}
	if err := Validate(servers); err != nil {
		return err
	}
	ocPath := filepath.Join(projectRoot, "opencode.json")
	existing, err := os.ReadFile(ocPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	out, err := Merge(existing, servers)
	if err != nil {
		return err
	}
	return os.WriteFile(ocPath, out, 0o644)
}

// List returns source vs applied servers without secret values.
func List(projectRoot string) ([]ListItem, error) {
	servers, src, err := LoadSources(projectRoot)
	if err != nil {
		return nil, err
	}
	applied, err := readApplied(projectRoot)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var items []ListItem
	for name, s := range servers {
		_, ok := applied[name]
		items = append(items, ListItem{
			Name:    name,
			Source:  src[name],
			Enabled: s.Enabled,
			Applied: ok,
		})
		seen[name] = struct{}{}
	}
	for name, s := range applied {
		if _, ok := seen[name]; ok {
			continue
		}
		items = append(items, ListItem{
			Name:    name,
			Source:  "",
			Enabled: s.Enabled,
			Applied: true,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func readApplied(projectRoot string) (map[string]Server, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, "opencode.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Server{}, nil
		}
		return nil, err
	}
	var doc struct {
		MCP map[string]json.RawMessage `json:"mcp"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse opencode.json: %w", err)
	}
	out := map[string]Server{}
	for name, body := range doc.MCP {
		s, err := parseServer(body)
		if err != nil {
			return nil, fmt.Errorf("opencode.json mcp %q: %w", name, err)
		}
		out[name] = s
	}
	return out, nil
}

const exampleConfig = `// Org MCP servers for OpenCode.
// Edit, then run: squad-oc mcp apply
// Use ${VAR} or {env:VAR} — never commit tokens (sk-, ghp_).
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
      },
      "enabled": false
    },
    "example-remote": {
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${MCP_TOKEN}"
      },
      "enabled": false
    }
  }
}
`

// InitExample writes a commented example into the resolved team dir if missing.
func InitExample(projectRoot string) (created bool, path string, err error) {
	path = OrgPath(projectRoot)
	if fileExists(path) {
		return false, path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, path, err
	}
	if err := os.WriteFile(path, []byte(exampleConfig), 0o644); err != nil {
		return false, path, err
	}
	return true, path, nil
}

// AppliedMissing reports org server names not present in opencode.json.
func AppliedMissing(projectRoot string) (missing []string, err error) {
	path := OrgPath(projectRoot)
	if !fileExists(path) {
		return nil, nil
	}
	org, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	applied, err := readApplied(projectRoot)
	if err != nil {
		return nil, err
	}
	for name := range org {
		if _, ok := applied[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func stripJSONC(data []byte) ([]byte, error) {
	var b strings.Builder
	b.Grow(len(data))
	inStr := false
	esc := false
	i := 0
	for i < len(data) {
		c := data[i]
		if inStr {
			b.WriteByte(c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			i++
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(data) {
			switch data[i+1] {
			case '/':
				i += 2
				for i < len(data) && data[i] != '\n' {
					i++
				}
				continue
			case '*':
				i += 2
				for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
					i++
				}
				if i+1 >= len(data) {
					return nil, fmt.Errorf("unterminated block comment")
				}
				i += 2
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return []byte(b.String()), nil
}
