package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xeaser/squad-opencode/internal/mcpconfig"
	"github.com/xeaser/squad-opencode/internal/share"
	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestRunChecksNotInitialized(t *testing.T) {
	root := t.TempDir()
	checks := RunChecks(root)
	var initOK bool
	for _, c := range checks {
		if c.Name == "Squad initialized" {
			initOK = c.OK
		}
	}
	if initOK {
		t.Fatal("expected not initialized")
	}
}

func TestRunChecksAfterInit(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root, Preset: "default"}); err != nil {
		t.Fatal(err)
	}
	checks := RunChecks(root)
	for _, name := range []string{"Squad initialized", "OpenCode squad agent", "Team file"} {
		found := false
		for _, c := range checks {
			if c.Name == name {
				found = true
				if !c.OK {
					t.Errorf("%s not ok: %s", name, c.Detail)
				}
			}
		}
		if !found {
			t.Errorf("missing check %s", name)
		}
	}
}

func TestMCPApplySoftFailWhenNotApplied(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	org := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "TOKEN": "${GITHUB_TOKEN}" }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(root, ".squad", "mcp-config.json"), org, 0o644); err != nil {
		t.Fatal(err)
	}
	c := findCheck(t, RunChecks(root), "MCP apply")
	if c.OK {
		t.Fatalf("expected soft fail before apply: %+v", c)
	}
	if c.Hard {
		t.Fatal("MCP apply must be soft (Hard: false)")
	}

	if err := mcpconfig.Apply(root); err != nil {
		t.Fatal(err)
	}
	c = findCheck(t, RunChecks(root), "MCP apply")
	if !c.OK {
		t.Fatalf("expected ok after apply: %+v", c)
	}
	if c.Hard {
		t.Fatal("MCP apply must stay soft")
	}
}

func TestMarketplaceSoftFailWhenUnresolved(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	c := findCheck(t, RunChecks(root), "Marketplaces")
	if !c.OK || c.Hard {
		t.Fatalf("no marketplaces should be soft ok: %+v", c)
	}

	if err := share.AddMarketplace(root, "missing", filepath.Join(root, "no-such-pack")); err != nil {
		t.Fatal(err)
	}
	c = findCheck(t, RunChecks(root), "Marketplaces")
	if c.OK {
		t.Fatalf("expected soft fail for unresolved path: %+v", c)
	}
	if c.Hard {
		t.Fatal("marketplaces must be soft (Hard: false)")
	}

	pack := t.TempDir()
	if err := share.AddMarketplace(root, "missing", pack); err != nil {
		t.Fatal(err)
	}
	c = findCheck(t, RunChecks(root), "Marketplaces")
	if !c.OK {
		t.Fatalf("expected ok after path exists: %+v", c)
	}
	if c.Hard {
		t.Fatal("marketplaces must stay soft")
	}
}

func findCheck(t *testing.T, checks []Check, name string) Check {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("missing check %s", name)
	return Check{}
}
