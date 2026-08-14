package watch

import (
	"context"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestBuildContextAndPass(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	issues := []Issue{{Number: 7, Title: "Bug", State: "OPEN"}}
	ctxText, err := BuildContext(root, issues)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctxText, "#7") || !strings.Contains(ctxText, "Lead") {
		t.Fatal(ctxText)
	}

	fake := &opencodeclient.FakeRunner{}
	ok, summary, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     true,
		Lister:      StaticLister{Issues: issues},
		Runner:      fake,
	})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(fake.Calls) != 1 {
		t.Fatal(fake.Calls)
	}
	if !strings.Contains(summary, "issues=1") {
		t.Fatal(summary)
	}

	_, triage, err := Pass(context.Background(), Options{
		ProjectRoot: root,
		Execute:     false,
		Lister:      StaticLister{Issues: issues},
	})
	if err != nil || !strings.Contains(triage, "#7") {
		t.Fatalf("%v %s", err, triage)
	}
}

func TestLoopOnce(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	err := Loop(context.Background(), Options{
		ProjectRoot: root,
		Once:        true,
		Lister:      StaticLister{Issues: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
}
