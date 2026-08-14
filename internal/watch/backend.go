package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// StateBackend persists Health across process restarts.
type StateBackend interface {
	Load(ctx context.Context) (Health, error)
	Save(ctx context.Context, h Health) error
}

// MemoryBackend holds Health in process (tests). Default CLI persistence is ralph-status.json.
type MemoryBackend struct{ H Health }

// Load implements StateBackend.
func (m *MemoryBackend) Load(context.Context) (Health, error) { return m.H, nil }

// Save implements StateBackend.
func (m *MemoryBackend) Save(_ context.Context, h Health) error {
	m.H = h
	return nil
}

// GitNotesBackend stores Health as a git note on HEAD.
type GitNotesBackend struct {
	Dir string
	Ref string // default refs/notes/squad-oc.ralph
}

func (g GitNotesBackend) ref() string {
	if g.Ref == "" {
		return "refs/notes/squad-oc.ralph"
	}
	return g.Ref
}

// Load implements StateBackend. Missing note starts fresh.
func (g GitNotesBackend) Load(ctx context.Context) (Health, error) {
	if err := requireGitRepo(ctx, g.Dir); err != nil {
		return Health{}, err
	}
	out, err := gitOutput(ctx, g.Dir, "notes", "--ref="+g.ref(), "show", "HEAD")
	if err != nil {
		return Health{}, nil
	}
	return unmarshalHealth(out)
}

// Save implements StateBackend.
func (g GitNotesBackend) Save(ctx context.Context, h Health) error {
	if err := requireGitRepo(ctx, g.Dir); err != nil {
		return err
	}
	data, err := marshalHealth(h)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "ralph-status-*.json")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_, err = gitOutput(ctx, g.Dir, "notes", "--ref="+g.ref(), "add", "-f", "-F", path, "HEAD")
	return err
}

// OrphanBranchBackend stores ralph-status.json on an orphan branch (no checkout).
type OrphanBranchBackend struct {
	Dir    string
	Branch string // default squad-oc/ralph-state
}

func (o OrphanBranchBackend) branch() string {
	if o.Branch == "" {
		return "squad-oc/ralph-state"
	}
	return o.Branch
}

// Load implements StateBackend. Missing branch starts fresh.
func (o OrphanBranchBackend) Load(ctx context.Context) (Health, error) {
	if err := requireGitRepo(ctx, o.Dir); err != nil {
		return Health{}, err
	}
	out, err := gitOutput(ctx, o.Dir, "show", o.branch()+":ralph-status.json")
	if err != nil {
		return Health{}, nil
	}
	return unmarshalHealth(out)
}

// Save implements StateBackend via hash-object + update-ref (worktree untouched).
func (o OrphanBranchBackend) Save(ctx context.Context, h Health) error {
	if err := requireGitRepo(ctx, o.Dir); err != nil {
		return err
	}
	data, err := marshalHealth(h)
	if err != nil {
		return err
	}
	blob, err := gitStdin(ctx, o.Dir, data, "hash-object", "-w", "--stdin")
	if err != nil {
		return err
	}
	treeIn := fmt.Sprintf("100644 blob %s\tralph-status.json\n", strings.TrimSpace(blob))
	tree, err := gitStdin(ctx, o.Dir, []byte(treeIn), "mktree")
	if err != nil {
		return err
	}
	tree = strings.TrimSpace(tree)
	args := []string{"commit-tree", tree, "-m", "ralph-status"}
	if parent, err := gitOutput(ctx, o.Dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+o.branch()); err == nil {
		parent = strings.TrimSpace(parent)
		if parent != "" {
			args = []string{"commit-tree", tree, "-p", parent, "-m", "ralph-status"}
		}
	}
	commit, err := gitOutput(ctx, o.Dir, args...)
	if err != nil {
		return err
	}
	_, err = gitOutput(ctx, o.Dir, "update-ref", "refs/heads/"+o.branch(), strings.TrimSpace(commit))
	return err
}

// ParseStateBackend maps memory|git-notes|orphan-branch. Empty/memory is nil (local file).
func ParseStateBackend(kind, dir string) (StateBackend, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "memory":
		return nil, nil
	case "git-notes":
		return GitNotesBackend{Dir: dir}, nil
	case "orphan-branch":
		return OrphanBranchBackend{Dir: dir}, nil
	default:
		return nil, fmt.Errorf("invalid --state-backend %q (want memory|git-notes|orphan-branch)", kind)
	}
}

func marshalHealth(h Health) ([]byte, error) {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func unmarshalHealth(raw string) (Health, error) {
	var h Health
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return Health{}, err
	}
	return h, nil
}

func requireGitRepo(ctx context.Context, dir string) error {
	if _, err := gitOutput(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("state backend requires a git repository")
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func gitStdin(ctx context.Context, dir string, stdin []byte, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
