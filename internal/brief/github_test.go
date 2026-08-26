package brief

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestGitHubParsesLists(t *testing.T) {
	g := GitHub{Dir: t.TempDir(), run: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "pr list") && strings.Contains(joined, "open"):
			return []byte(`[{"number":3,"title":"WIP","author":{"login":"ann"},"isDraft":true,"reviewDecision":"REVIEW_REQUIRED"}]`), nil
		case strings.Contains(joined, "pr list") && strings.Contains(joined, "merged"):
			return []byte(`[{"number":1,"title":"done","mergedAt":"2026-08-01T00:00:00Z"}]`), nil
		case strings.Contains(joined, "issue list"):
			return []byte(`[{"number":11,"title":"unstarted","createdAt":"2026-06-01T00:00:00Z"}]`), nil
		case strings.Contains(joined, "pr view"):
			// Live gh pr view --json closingIssuesReferences is an array, not {nodes:[]}.
			return []byte(`{"closingIssuesReferences":[{"number":10}],"body":""}`), nil
		default:
			return nil, fmt.Errorf("unexpected gh %v", args)
		}
	}}
	prs := GitHubPRs{GitHub: g}
	open, err := prs.ListOpen(context.Background())
	if err != nil || len(open) != 1 || open[0].Number != 3 || !open[0].Draft || open[0].Author != "ann" {
		t.Fatalf("open %+v %v", open, err)
	}
	if len(open[0].LinkedIssue) != 1 || open[0].LinkedIssue[0] != 10 {
		t.Fatalf("linked %+v", open[0].LinkedIssue)
	}
	tix := GitHubTickets{GitHub: g}
	issues, err := tix.ListOpen(context.Background())
	if err != nil || len(issues) != 1 || issues[0].Number != 11 {
		t.Fatalf("issues %+v %v", issues, err)
	}
	merged, err := prs.ListMerged(context.Background(), 5)
	if err != nil || len(merged) != 1 || merged[0].Number != 1 {
		t.Fatalf("merged %+v %v", merged, err)
	}
}

func TestGitHubClosingIssuesBodyFallback(t *testing.T) {
	g := GitHub{Dir: t.TempDir(), run: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "pr list") && strings.Contains(joined, "open"):
			return []byte(`[{"number":4,"title":"body-only","author":{"login":"bob"},"isDraft":false,"reviewDecision":""}]`), nil
		case strings.Contains(joined, "pr view"):
			return []byte(`{"closingIssuesReferences":[],"body":"Closes #10"}`), nil
		default:
			return nil, fmt.Errorf("unexpected gh %v", args)
		}
	}}
	open, err := GitHubPRs{GitHub: g}.ListOpen(context.Background())
	if err != nil || len(open) != 1 {
		t.Fatalf("open %+v %v", open, err)
	}
	if len(open[0].LinkedIssue) != 1 || open[0].LinkedIssue[0] != 10 {
		t.Fatalf("body fallback linked %+v", open[0].LinkedIssue)
	}
}

func TestGitHubRunErrorIsError(t *testing.T) {
	g := GitHub{run: func(context.Context, ...string) ([]byte, error) {
		return nil, fmt.Errorf("gh not found")
	}}
	_, err := GitHubPRs{GitHub: g}.ListOpen(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
