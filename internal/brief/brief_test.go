package brief

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	assertJSONEmptyArrays(t, raw)

	empty, err := Collect(context.Background(), Options{
		ProjectRoot: root,
		Tickets:     fakeTickets{},
		PRs:         fakePRs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !empty.PRs.OK || !empty.Tickets.OK {
		t.Fatalf("empty sources should succeed: prs=%+v tickets=%+v", empty.PRs, empty.Tickets)
	}
	rawEmpty, err := FormatJSON(empty)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEmptyArrays(t, rawEmpty)
}

func assertJSONEmptyArrays(t *testing.T, raw []byte) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	paths := [][]string{
		{"team", "members"},
		{"prs", "items"},
		{"tickets", "items"},
		{"inProgress", "prs"},
		{"inProgress", "designReviews"},
		{"lastDone", "prs"},
		{"needsYou"},
	}
	for _, path := range paths {
		cur := any(m)
		for _, key := range path {
			obj, ok := cur.(map[string]any)
			if !ok {
				t.Fatalf("%v not an object in %s", path, raw)
			}
			cur = obj[key]
		}
		if cur == nil {
			t.Fatalf("%v is null, want [] in %s", path, raw)
		}
		if _, ok := cur.([]any); !ok {
			t.Fatalf("%v is %T, want array in %s", path, cur, raw)
		}
	}
}

func TestCollectLocalCommsRalphReviews(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	comms := filepath.Join(squad.ResolveDir(root), "comms")
	review := `## Handoff: tester → lead

## Review

- **Verdict:** reject
- **Author:** backend
- **Fix owner:** backend
- **Reasons:** tests missing
`
	if err := os.WriteFile(filepath.Join(comms, "2026-08-26-tester-to-lead.md"), []byte(review), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comms, "2026-08-26-auth-design-review.md"), []byte("## Design Review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	accept := `## Review

- **Verdict:** accept
- **Author:** frontend
- **Fix owner:** lead
`
	if err := os.WriteFile(filepath.Join(comms, "2026-08-26-ok.md"), []byte(accept), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(squad.ResolveDir(root), "ralph-status.json"), []byte(`{
  "lastSummary": "polled 2 issues",
  "lastError": "timeout",
  "overnight": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(squad.ResolveDir(root), "ralph-stop"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Collect(context.Background(), Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if !containsName(rep.InProgress.DesignReviews, "2026-08-26-auth-design-review.md") {
		t.Fatalf("design reviews: %v", rep.InProgress.DesignReviews)
	}
	if len(rep.NeedsYou) != 1 || !rep.NeedsYou[0].SameOwner {
		t.Fatalf("needs you: %+v", rep.NeedsYou)
	}
	if !rep.Ralph.Present || !strings.Contains(rep.Ralph.LastSummary, "polled") || !rep.Ralph.Stop || !rep.Ralph.Overnight {
		t.Fatalf("ralph: %+v", rep.Ralph)
	}
	text := Format(rep)
	if !strings.Contains(text, "FLAG author==fix-owner") {
		t.Fatalf("flag missing:\n%s", text)
	}
}

func containsName(paths []string, name string) bool {
	for _, p := range paths {
		if strings.HasSuffix(p, name) {
			return true
		}
	}
	return false
}

type fakeTickets struct{ items []Ticket }

func (f fakeTickets) ListOpen(context.Context) ([]Ticket, error) { return f.items, nil }

type fakePRs struct {
	open      []PR
	merged    []PR
	openErr   error
	mergedErr error
}

func (f fakePRs) ListOpen(context.Context) ([]PR, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f.open, nil
}

func (f fakePRs) ListMerged(context.Context, int) ([]PR, error) {
	if f.mergedErr != nil {
		return nil, f.mergedErr
	}
	return f.merged, nil
}

func TestCollectLastDoneUnavailableWhenMergedFails(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	rep, err := Collect(context.Background(), Options{
		ProjectRoot: root,
		Tickets:     fakeTickets{},
		PRs: fakePRs{
			open:      []PR{{Number: 1, Title: "open"}},
			mergedErr: fmt.Errorf("merged list failed"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.PRs.OK {
		t.Fatalf("open should succeed: %+v", rep.PRs)
	}
	text := Format(rep)
	sec := section(text, "Last done")
	if !strings.Contains(sec, "unavailable") {
		t.Fatalf("want Last done unavailable, got %q\n%s", sec, text)
	}
	if strings.Contains(sec, "(none)") {
		t.Fatalf("Last done must not be (none) when ListMerged fails:\n%s", text)
	}
}

func section(text, heading string) string {
	idx := strings.Index(text, heading)
	if idx < 0 {
		return ""
	}
	rest := text[idx:]
	if next := strings.Index(rest[len(heading):], "\n\n"); next >= 0 {
		return rest[:len(heading)+next]
	}
	return rest
}

func TestCollectGitHubNextSkipsLinked(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rep, err := Collect(context.Background(), Options{
		ProjectRoot: root,
		Tickets: fakeTickets{[]Ticket{
			{Source: "github", ID: "#10", Number: 10, Title: "has PR", CreatedAt: old},
			{Source: "github", ID: "#11", Number: 11, Title: "unstarted", CreatedAt: newer},
		}},
		PRs: fakePRs{
			open:   []PR{{Number: 3, Title: "fix 10", Author: "a", LinkedIssue: []int{10}}},
			merged: []PR{{Number: 2, Title: "shipped", MergedAt: "2026-08-01T00:00:00Z"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.PRs.OK || len(rep.PRs.Items) != 1 || !rep.Tickets.OK {
		t.Fatalf("%+v", rep)
	}
	if rep.Next == nil || rep.Next.Number != 11 {
		t.Fatalf("next %+v", rep.Next)
	}
	if len(rep.LastDone.PRs) != 1 || rep.LastDone.PRs[0].Number != 2 {
		t.Fatalf("last %+v", rep.LastDone)
	}
	text := Format(rep)
	if strings.Contains(text, "unavailable") {
		t.Fatalf("sources ok should not say unavailable:\n%s", text)
	}
	if !strings.Contains(text, "#11") || !strings.Contains(text, "unstarted") {
		t.Fatalf("next missing:\n%s", text)
	}
}

func TestPickNext(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("oldest unlinked wins", func(t *testing.T) {
		got := pickNext([]Ticket{
			{ID: "#2", Number: 2, Title: "newer", CreatedAt: t2},
			{ID: "#1", Number: 1, Title: "older", CreatedAt: t1},
		}, nil)
		if got == nil || got.Number != 1 {
			t.Fatalf("want #1, got %+v", got)
		}
	})

	t.Run("skip linked issue numbers", func(t *testing.T) {
		got := pickNext([]Ticket{
			{ID: "#1", Number: 1, Title: "linked", CreatedAt: t1},
			{ID: "#2", Number: 2, Title: "free", CreatedAt: t2},
		}, []PR{{Number: 10, LinkedIssue: []int{1}}})
		if got == nil || got.Number != 2 {
			t.Fatalf("want #2 (unlinked), got %+v", got)
		}
	})

	t.Run("tie-break CreatedAt then Number", func(t *testing.T) {
		got := pickNext([]Ticket{
			{ID: "#5", Number: 5, Title: "higher", CreatedAt: t1},
			{ID: "#3", Number: 3, Title: "lower", CreatedAt: t1},
		}, nil)
		if got == nil || got.Number != 3 {
			t.Fatalf("want #3 on equal CreatedAt, got %+v", got)
		}
	})

	t.Run("parse number from ID when Number is zero", func(t *testing.T) {
		got := pickNext([]Ticket{
			{ID: "#7", Title: "from-id", CreatedAt: t1},
			{ID: "#8", Number: 8, Title: "newer", CreatedAt: t2},
		}, []PR{{LinkedIssue: []int{7}}})
		if got == nil || got.ID != "#8" {
			t.Fatalf("want #8 after skipping ID-parsed #7, got %+v", got)
		}
	})

	t.Run("bad ID scanf yields zero number not linked skip", func(t *testing.T) {
		got := pickNext([]Ticket{
			{ID: "nope", Title: "unparseable", CreatedAt: t2},
			{ID: "#1", Number: 1, Title: "other", CreatedAt: t1},
		}, []PR{{LinkedIssue: []int{1}}})
		if got == nil || got.ID != "nope" {
			t.Fatalf("want unparseable ticket when only linked alternative, got %+v", got)
		}
	})
}
