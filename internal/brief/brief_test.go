package brief

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestCollectNotInitialized(t *testing.T) {
	_, err := Collect(context.Background(), Options{ProjectRoot: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("got %v", err)
	}
}

func TestCollectTeamAndJSONKeys(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{
		ProjectRoot:        root,
		ProjectDescription: "brief team",
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := Collect(context.Background(), Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Team.Theme != "none" {
		t.Fatalf("theme %q", rep.Team.Theme)
	}
	if len(rep.Team.Members) < 1 {
		t.Fatalf("members %+v", rep.Team.Members)
	}
	if rep.PRs.OK || rep.Tickets.OK {
		t.Fatal("nil sources must be unavailable")
	}
	if rep.PRs.Error == "" || rep.Tickets.Error == "" {
		t.Fatal("want unavailable reason")
	}
	if !rep.Ceremonies.Present || !strings.Contains(rep.Ceremonies.Path, "ceremonies.md") {
		t.Fatalf("ceremonies %+v", rep.Ceremonies)
	}

	text := Format(rep)
	for _, want := range []string{"Morning brief", "Team", "Open PRs", "Tickets", "In progress", "Last done", "Next", "Ralph", "Needs you", "unavailable"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}

	raw, err := FormatJSON(rep)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"team", "prs", "tickets", "inProgress", "lastDone", "next", "ralph", "needsYou", "ceremonies"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("json missing %s: %s", k, raw)
		}
	}
}
