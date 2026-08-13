package doctor

import (
	"testing"

	"github.com/squad-opencode/squad-opencode/internal/squad"
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
