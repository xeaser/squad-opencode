package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestCopilotShapeToOpenCode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "org-mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	servers, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gh, ok := servers["github"]
	if !ok {
		t.Fatal("missing github")
	}
	if gh.Type != "local" {
		t.Fatalf("type %s", gh.Type)
	}
	wantCmd := []string{"npx", "-y", "@modelcontextprotocol/server-github"}
	if !reflect.DeepEqual(gh.Command, wantCmd) {
		t.Fatalf("command %v", gh.Command)
	}
	if gh.Environment["GITHUB_PERSONAL_ACCESS_TOKEN"] != "{env:GITHUB_TOKEN}" {
		t.Fatalf("env %v", gh.Environment)
	}
	if !gh.Enabled {
		t.Fatal("expected enabled")
	}
	docs, ok := servers["docs"]
	if !ok {
		t.Fatal("missing docs")
	}
	if docs.Type != "remote" || docs.URL != "https://example.com/mcp" || docs.Enabled {
		t.Fatalf("docs %+v", docs)
	}
	if docs.Headers["Authorization"] != "Bearer {env:DOCS_TOKEN}" {
		t.Fatalf("headers %v", docs.Headers)
	}

	existing := []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n")
	out, err := Merge(existing, servers)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "expected-opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, out, want) {
		t.Fatalf("merge snippet\ngot  %s\nwant %s", out, want)
	}
}

func TestOpenCodePassthrough(t *testing.T) {
	in := []byte(`{
  "mcp": {
    "github": {
      "type": "local",
      "command": ["npx", "-y", "@modelcontextprotocol/server-github"],
      "enabled": true,
      "environment": { "GITHUB_PERSONAL_ACCESS_TOKEN": "{env:GITHUB_TOKEN}" }
    }
  }
}`)
	servers, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	gh := servers["github"]
	if gh.Type != "local" || !gh.Enabled {
		t.Fatalf("%+v", gh)
	}
	if !reflect.DeepEqual(gh.Command, []string{"npx", "-y", "@modelcontextprotocol/server-github"}) {
		t.Fatalf("command %v", gh.Command)
	}
	if gh.Environment["GITHUB_PERSONAL_ACCESS_TOKEN"] != "{env:GITHUB_TOKEN}" {
		t.Fatalf("env %v", gh.Environment)
	}
}

func TestMergePreservesSchema(t *testing.T) {
	existing := []byte(`{
  "$schema": "https://opencode.ai/config.json",
  "provider": { "xai": { "options": {} } },
  "model": "xai/grok"
}`)
	servers := map[string]Server{
		"github": {
			Type:    "local",
			Command: []string{"npx", "-y", "@modelcontextprotocol/server-github"},
			Enabled: true,
		},
	}
	out, err := Merge(existing, servers)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["$schema"] != "https://opencode.ai/config.json" {
		t.Fatalf("dropped $schema: %s", out)
	}
	if _, ok := doc["provider"]; !ok {
		t.Fatalf("dropped provider: %s", out)
	}
	if doc["model"] != "xai/grok" {
		t.Fatalf("dropped model: %s", out)
	}
}

func TestEnvVarRewrite(t *testing.T) {
	in := []byte(`{
  "mcpServers": {
    "svc": {
      "command": "npx",
      "args": ["-y", "${PKG}"],
      "env": { "TOKEN": "${FOO}", "ALREADY": "{env:BAR}" }
    },
    "remote": {
      "url": "https://example.com/${TENANT}/mcp"
    }
  }
}`)
	servers, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	env := servers["svc"].Environment
	if env["TOKEN"] != "{env:FOO}" {
		t.Fatalf("TOKEN %q", env["TOKEN"])
	}
	if env["ALREADY"] != "{env:BAR}" {
		t.Fatalf("ALREADY %q", env["ALREADY"])
	}
	if !reflect.DeepEqual(servers["svc"].Command, []string{"npx", "-y", "{env:PKG}"}) {
		t.Fatalf("command %v", servers["svc"].Command)
	}
	if servers["remote"].URL != "https://example.com/{env:TENANT}/mcp" {
		t.Fatalf("url %q", servers["remote"].URL)
	}
}

func TestRejectHardcodedTokens(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".squad"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "demo"],
      "env": { "TOKEN": "sk-secretvalue" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	err := Apply(root)
	if err == nil {
		t.Fatal("expected error for sk- token")
	}
	if !strings.Contains(err.Error(), "sk-") {
		t.Fatalf("error should mention sk-: %v", err)
	}

	body = []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "demo"],
      "env": { "TOKEN": "ghp_secretvalue" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	err = Apply(root)
	if err == nil {
		t.Fatal("expected error for ghp_ token")
	}
	if !strings.Contains(err.Error(), "ghp_") {
		t.Fatalf("error should mention ghp_: %v", err)
	}

	for _, tc := range []struct {
		val string
		tok string
	}{
		{"ApiKey=sk-secretvalue", "sk-"},
		{"sk-secretvalue{env:FOO}", "sk-"},
		{"token=ghp_secretvalue", "ghp_"},
		{"Bearer sk-secretvalue", "sk-"},
	} {
		body = []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "demo"],
      "env": { "TOKEN": "` + tc.val + `" }
    }
  }
}`)
		if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), body, 0o644); err != nil {
			t.Fatal(err)
		}
		err = Apply(root)
		if err == nil {
			t.Fatalf("expected error for %q", tc.val)
		}
		if !strings.Contains(err.Error(), tc.tok) {
			t.Fatalf("%q: error should mention %s: %v", tc.val, tc.tok, err)
		}
	}
}

func TestOrgWinsSameName(t *testing.T) {
	existing := []byte(`{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "github": {
      "type": "local",
      "command": ["old"],
      "enabled": true
    },
    "keep": {
      "type": "remote",
      "url": "https://keep.example",
      "enabled": true
    }
  }
}`)
	servers := map[string]Server{
		"github": {
			Type:    "local",
			Command: []string{"npx", "-y", "new"},
			Enabled: true,
		},
	}
	out, err := Merge(existing, servers)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	mcp, _ := doc["mcp"].(map[string]any)
	gh, _ := mcp["github"].(map[string]any)
	cmd, _ := gh["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" || cmd[2] != "new" {
		t.Fatalf("org should win: %s", out)
	}
	if _, ok := mcp["keep"]; !ok {
		t.Fatalf("unrelated server dropped: %s", out)
	}
}

func TestApplyReadsLinkedTeam(t *testing.T) {
	service := t.TempDir()
	shared := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: service, ProjectDescription: "svc"}); err != nil {
		t.Fatal(err)
	}
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: shared, ProjectDescription: "shared"}); err != nil {
		t.Fatal(err)
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(squad.SquadDir(shared), "mcp-config.json"), org, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(squad.SquadDir(service), "mcp-config.json"), []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target, err := squad.ResolveLinkTarget(shared)
	if err != nil {
		t.Fatal(err)
	}
	if err := squad.SetLink(service, target); err != nil {
		t.Fatal(err)
	}
	if err := Apply(service); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(service, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mcp, _ := doc["mcp"].(map[string]any)
	if _, ok := mcp["github"]; !ok {
		t.Fatalf("linked org MCP not applied: %s", raw)
	}
}

func TestApplyReadsOnlyOrgPath(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "from-org"]
    }
  }
}`)
	cwd := []byte(`{
  "mcpServers": {
    "packonly": {
      "command": "npx",
      "args": ["-y", "from-cwd"]
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), org, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mcp-config.json"), cwd, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mcp, _ := doc["mcp"].(map[string]any)
	if _, ok := mcp["github"]; !ok {
		t.Fatalf("org server missing: %s", raw)
	}
	if _, ok := mcp["packonly"]; ok {
		t.Fatalf("cwd mcp-config.json must not be an apply source: %s", raw)
	}
}

func TestInitExampleParses(t *testing.T) {
	servers, err := Parse([]byte(exampleConfig))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["github"]; !ok {
		t.Fatalf("example missing github: %v", servers)
	}
	if _, ok := servers["example-remote"]; !ok {
		t.Fatalf("example missing example-remote: %v", servers)
	}
	if servers["github"].Enabled {
		t.Fatal("example github should be disabled")
	}
}

func TestJSONCCommentsStrip(t *testing.T) {
	in := []byte(`
// line comment
{
  /* block
     comment */
  "mcpServers": {
    "svc": {
      "command": "npx"
    }
  }
}
`)
	servers, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["svc"]; !ok {
		t.Fatalf("%v", servers)
	}
	_, err = Parse([]byte(`{"mcpServers": {"svc": {"command": "npx"},}}`))
	if err == nil {
		t.Fatal("expected trailing comma to fail parse")
	}
}

func TestInitExampleMissingOnly(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	created, path, err := InitExample(root)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"keep":{"command":"npx"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	created, _, err = InitExample(root)
	if err != nil || created {
		t.Fatalf("second init should not overwrite: created=%v err=%v", created, err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(got), "keep") {
		t.Fatalf("overwrote example: %s", got)
	}
}

func TestListOmitsSecrets(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "demo"],
      "env": { "TOKEN": "${GITHUB_TOKEN}" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), org, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(root); err != nil {
		t.Fatal(err)
	}
	items, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	blob, _ := json.Marshal(items)
	if strings.Contains(string(blob), "sk-") || strings.Contains(string(blob), "ghp_") {
		t.Fatalf("list leaked a token prefix: %s", blob)
	}
	if strings.Contains(strings.ToLower(string(blob)), "github_token") && strings.Contains(string(blob), "${") {
		t.Fatalf("list should not print env values: %s", blob)
	}
	found := false
	for _, it := range items {
		if it.Name == "github" && it.Applied && it.Enabled && it.Source == "org" {
			found = true
		}
	}
	if !found {
		t.Fatalf("list rows: %+v", items)
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var da, db any
	if err := json.Unmarshal(a, &da); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &db); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(da, db)
}
