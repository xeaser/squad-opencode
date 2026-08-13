package githubissues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/squad-opencode/squad-opencode/internal/watch"
)

// ParseListJSON parses `gh issue list --json number,title,state`.
func ParseListJSON(data []byte) ([]watch.Issue, error) {
	var issues []watch.Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// GHLister runs gh issue list.
type GHLister struct {
	Dir string
}

// List implements watch.IssueLister.
func (g GHLister) List(ctx context.Context) ([]watch.Issue, error) {
	cmd := exec.CommandContext(ctx, "gh", "issue", "list", "--json", "number,title,state", "--limit", "20")
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
