package brief

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type GitHub struct {
	Dir string
	run func(ctx context.Context, args ...string) ([]byte, error)
}

type GitHubPRs struct{ GitHub }
type GitHubTickets struct{ GitHub }

func (g GitHub) exec(ctx context.Context, args ...string) ([]byte, error) {
	if g.run != nil {
		return g.run(ctx, args...)
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh not found")
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	if g.Dir != "" {
		cmd.Dir = g.Dir
	}
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "GH_FORCE_TTY=0", "GH_PROMPT_DISABLED=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func (g GitHubPRs) ListOpen(ctx context.Context) ([]PR, error) {
	raw, err := g.exec(ctx, "pr", "list", "--state", "open", "--json", "number,title,author,isDraft,reviewDecision")
	if err != nil {
		return nil, err
	}
	prs, err := parsePRs(raw)
	if err != nil {
		return nil, err
	}
	for i := range prs {
		prs[i].LinkedIssue = g.closingIssues(ctx, prs[i].Number)
	}
	return prs, nil
}

func (g GitHubPRs) ListMerged(ctx context.Context, limit int) ([]PR, error) {
	if limit <= 0 {
		limit = 5
	}
	raw, err := g.exec(ctx, "pr", "list", "--state", "merged", "--limit", strconv.Itoa(limit), "--json", "number,title,mergedAt")
	if err != nil {
		return nil, err
	}
	return parsePRs(raw)
}

func (g GitHubTickets) ListOpen(ctx context.Context) ([]Ticket, error) {
	raw, err := g.exec(ctx, "issue", "list", "--state", "open", "--json", "number,title,createdAt")
	if err != nil {
		return nil, err
	}
	return parseTickets(raw)
}

func (g GitHubPRs) closingIssues(ctx context.Context, n int) []int {
	raw, err := g.exec(ctx, "pr", "view", strconv.Itoa(n), "--json", "closingIssuesReferences,body")
	if err != nil || len(raw) == 0 {
		return nil
	}
	var wrap struct {
		Body                    string `json:"body"`
		ClosingIssuesReferences struct {
			Nodes []struct {
				Number int `json:"number"`
			} `json:"nodes"`
		} `json:"closingIssuesReferences"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return parseBodyIssueRefs(string(raw))
	}
	var out []int
	for _, node := range wrap.ClosingIssuesReferences.Nodes {
		if node.Number > 0 {
			out = append(out, node.Number)
		}
	}
	if len(out) == 0 {
		return parseBodyIssueRefs(wrap.Body)
	}
	return out
}

func parsePRs(raw []byte) ([]PR, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var rows []struct {
		Number         int    `json:"number"`
		Title          string `json:"title"`
		Author         any    `json:"author"`
		IsDraft        bool   `json:"isDraft"`
		ReviewDecision string `json:"reviewDecision"`
		MergedAt       string `json:"mergedAt"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]PR, 0, len(rows))
	for _, r := range rows {
		out = append(out, PR{
			Number:   r.Number,
			Title:    r.Title,
			Author:   authorLogin(r.Author),
			Draft:    r.IsDraft,
			Review:   r.ReviewDecision,
			MergedAt: r.MergedAt,
		})
	}
	return out, nil
}

func parseTickets(raw []byte) ([]Ticket, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var rows []struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"createdAt"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]Ticket, 0, len(rows))
	for _, r := range rows {
		out = append(out, Ticket{
			Source:    "github",
			ID:        fmt.Sprintf("#%d", r.Number),
			Number:    r.Number,
			Title:     r.Title,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func authorLogin(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, ok := t["login"].(string); ok {
			return s
		}
	}
	return ""
}

func parseBodyIssueRefs(body string) []int {
	low := strings.ToLower(body)
	var out []int
	for _, key := range []string{"closes #", "fixes #", "mentions #"} {
		rest := low
		for {
			i := strings.Index(rest, key)
			if i < 0 {
				break
			}
			rest = rest[i+len(key):]
			n := 0
			for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
				n = n*10 + int(rest[0]-'0')
				rest = rest[1:]
			}
			if n > 0 {
				out = append(out, n)
			}
		}
	}
	return out
}
