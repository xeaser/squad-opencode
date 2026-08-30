package githubissues

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestParseListJSON(t *testing.T) {
	raw := []byte(`[{"number":1,"title":"Hi","state":"OPEN"}]`)
	issues, err := ParseListJSON(raw)
	if err != nil || len(issues) != 1 || issues[0].Number != 1 {
		t.Fatalf("%v %+v", err, issues)
	}
}

func TestGHListerArgsLabel(t *testing.T) {
	g := GHLister{Labels: []string{"bug"}}
	got := strings.Join(g.args(), " ")
	if !strings.Contains(got, "--label bug") {
		t.Fatalf("want --label bug in %q", got)
	}
	if !strings.Contains(got, "--limit 20") {
		t.Fatalf("default limit 20 in %q", got)
	}
	if !strings.Contains(got, "labels") {
		t.Fatalf("want labels in --json: %q", got)
	}
}

func TestParseListJSONLabels(t *testing.T) {
	raw := []byte(`[{"number":1,"title":"Hi","state":"OPEN","labels":[{"name":"bug","id":"1"},{"name":"ralph-retry"}]}]`)
	issues, err := ParseListJSON(raw)
	if err != nil || len(issues) != 1 {
		t.Fatalf("%v %+v", err, issues)
	}
	if len(issues[0].Labels) != 2 || issues[0].Labels[0] != "bug" || issues[0].Labels[1] != "ralph-retry" {
		t.Fatalf("labels %+v", issues[0].Labels)
	}
}

func TestGHPRCheckerClosingRefsAndBody(t *testing.T) {
	g := GHPRChecker{Dir: t.TempDir(), run: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "pr list"):
			if !strings.Contains(joined, "--state open") {
				t.Fatalf("want open PRs only: %q", joined)
			}
			return []byte(`[{"number":3},{"number":4},{"number":5}]`), nil
		case strings.Contains(joined, "pr view 3"):
			return []byte(`{"closingIssuesReferences":[{"number":7}],"body":""}`), nil
		case strings.Contains(joined, "pr view 4"):
			return []byte(`{"closingIssuesReferences":[],"body":"Closes #8\nfixes #9\nmentions #10"}`), nil
		case strings.Contains(joined, "pr view 5"):
			return []byte(`{"closingIssuesReferences":[],"body":"see #11 and also #7"}`), nil
		default:
			return nil, fmt.Errorf("unexpected gh %v", args)
		}
	}}
	links, err := g.OpenLinks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if links[7] != 3 || links[8] != 4 || links[9] != 4 || links[10] != 4 {
		t.Fatalf("links %+v", links)
	}
	if _, ok := links[11]; ok {
		t.Fatalf("bare #N must not link: %+v", links)
	}
}

func TestGHPRCheckerFirstPRWins(t *testing.T) {
	g := GHPRChecker{run: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "pr list"):
			return []byte(`[{"number":3},{"number":9}]`), nil
		case strings.Contains(joined, "pr view 3"):
			return []byte(`{"closingIssuesReferences":[{"number":7}],"body":""}`), nil
		case strings.Contains(joined, "pr view 9"):
			return []byte(`{"closingIssuesReferences":[{"number":7}],"body":""}`), nil
		default:
			return nil, fmt.Errorf("unexpected gh %v", args)
		}
	}}
	links, err := g.OpenLinks(context.Background())
	if err != nil || links[7] != 3 {
		t.Fatalf("first PR wins: %+v %v", links, err)
	}
}

func TestGHPRCheckerExecError(t *testing.T) {
	g := GHPRChecker{run: func(context.Context, ...string) ([]byte, error) {
		return nil, fmt.Errorf("gh down")
	}}
	_, err := g.OpenLinks(context.Background())
	if err == nil {
		t.Fatal("want exec error")
	}
}
