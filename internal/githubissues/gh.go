package githubissues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xeaser/squad-opencode/internal/watch"
)

// ParseListJSON parses `gh issue list --json number,title,state,labels`.
func ParseListJSON(data []byte) ([]watch.Issue, error) {
	var rows []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	issues := make([]watch.Issue, 0, len(rows))
	for _, r := range rows {
		var labels []string
		for _, l := range r.Labels {
			if l.Name != "" {
				labels = append(labels, l.Name)
			}
		}
		issues = append(issues, watch.Issue{
			Number: r.Number,
			Title:  r.Title,
			State:  r.State,
			Labels: labels,
		})
	}
	return issues, nil
}

// GHLister runs `gh issue list` (already open issues only; no --assignee).
type GHLister struct {
	Dir    string
	Labels []string // --label, repeatable; passed to `gh issue list --label`
	Limit  int      // default 20
}

func (g GHLister) args() []string {
	limit := g.Limit
	if limit <= 0 {
		limit = 20
	}
	args := []string{"issue", "list", "--json", "number,title,state,labels", "--limit", strconv.Itoa(limit)}
	for _, label := range g.Labels {
		if label == "" {
			continue
		}
		args = append(args, "--label", label)
	}
	return args
}

// List implements watch.IssueLister.
func (g GHLister) List(ctx context.Context) ([]watch.Issue, error) {
	cmd := exec.CommandContext(ctx, "gh", g.args()...)
	if g.Dir != "" {
		cmd.Dir = g.Dir
	}
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "GH_FORCE_TTY=0", "GH_PROMPT_DISABLED=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh issue list: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return nil, nil
	}
	issues, err := ParseListJSON(out)
	if err != nil {
		preview := string(out)
		if len(preview) > 160 {
			preview = preview[:160]
		}
		return nil, fmt.Errorf("gh issue list: not JSON (%v): %q", err, preview)
	}
	return issues, nil
}

// GHPRChecker lists open PRs and maps linked issues (closing refs or body keywords).
type GHPRChecker struct {
	Dir string
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// OpenLinks implements watch.LinkedPRChecker. Issue → first matching open PR.
func (g GHPRChecker) OpenLinks(ctx context.Context) (map[int]int, error) {
	raw, err := g.exec(ctx, "pr", "list", "--state", "open", "--limit", "1000", "--json", "number")
	if err != nil {
		return nil, err
	}
	var prs []struct {
		Number int `json:"number"`
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &prs); err != nil {
			return nil, fmt.Errorf("gh pr list: not JSON (%w)", err)
		}
	}
	links := make(map[int]int)
	for _, pr := range prs {
		if pr.Number <= 0 {
			continue
		}
		issues, err := g.linkedIssues(ctx, pr.Number)
		if err != nil {
			return nil, err
		}
		for _, n := range issues {
			if n > 0 {
				if _, exists := links[n]; !exists {
					links[n] = pr.Number
				}
			}
		}
	}
	return links, nil
}

func (g GHPRChecker) linkedIssues(ctx context.Context, n int) ([]int, error) {
	raw, err := g.exec(ctx, "pr", "view", strconv.Itoa(n), "--json", "closingIssuesReferences,body")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var wrap struct {
		Body                    string `json:"body"`
		ClosingIssuesReferences []struct {
			Number int `json:"number"`
		} `json:"closingIssuesReferences"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return parseBodyIssueRefs(string(raw)), nil
	}
	var out []int
	seen := map[int]bool{}
	for _, ref := range wrap.ClosingIssuesReferences {
		if ref.Number > 0 && !seen[ref.Number] {
			seen[ref.Number] = true
			out = append(out, ref.Number)
		}
	}
	for _, num := range parseBodyIssueRefs(wrap.Body) {
		if num > 0 && !seen[num] {
			seen[num] = true
			out = append(out, num)
		}
	}
	return out, nil
}

func (g GHPRChecker) exec(ctx context.Context, args ...string) ([]byte, error) {
	if g.run != nil {
		return g.run(ctx, args...)
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

// GHProjectSource lists GitHub Project v2 items (issue number + Status).
type GHProjectSource struct {
	Dir string
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Items implements watch.ProjectSource.
func (g GHProjectSource) Items(ctx context.Context, project int) ([]watch.ProjectItem, error) {
	owner, err := g.owner(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := g.exec(ctx, "project", "item-list", strconv.Itoa(project), "--owner", owner, "--format", "json", "--limit", "1000")
	if err != nil {
		return nil, err
	}
	return ParseProjectItemsJSON(raw)
}

func (g GHProjectSource) owner(ctx context.Context) (string, error) {
	raw, err := g.exec(ctx, "repo", "view", "--json", "owner")
	if err != nil {
		return "", err
	}
	var wrap struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil || wrap.Owner.Login == "" {
		preview := string(raw)
		if len(preview) > 160 {
			preview = preview[:160]
		}
		if err != nil {
			return "", fmt.Errorf("gh repo view: not JSON (%w): %q", err, preview)
		}
		return "", fmt.Errorf("gh repo view: empty owner: %q", preview)
	}
	return wrap.Owner.Login, nil
}

func (g GHProjectSource) exec(ctx context.Context, args ...string) ([]byte, error) {
	if g.run != nil {
		return g.run(ctx, args...)
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

// ParseProjectItemsJSON parses `gh project item-list --format json`.
func ParseProjectItemsJSON(data []byte) ([]watch.ProjectItem, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var wrap struct {
		Items []struct {
			Status  string `json:"status"`
			Content struct {
				Number int `json:"number"`
			} `json:"content"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("gh project item-list: not JSON (%w)", err)
	}
	var out []watch.ProjectItem
	for _, it := range wrap.Items {
		if it.Content.Number <= 0 {
			continue
		}
		out = append(out, watch.ProjectItem{
			Number: it.Content.Number,
			Status: it.Status,
		})
	}
	return out, nil
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
