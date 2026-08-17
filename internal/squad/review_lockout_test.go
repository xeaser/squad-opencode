package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Lockout sentences that must appear in coordinator and reviewer host files.
const (
	lockoutSpawnSpecialists  = "must spawn specialists for multi-layer work"
	lockoutNoPatchRejected   = "must not patch rejected specialist output"
	lockoutRejectInHandoff   = "reject in a handoff file"
	lockoutNextNotAuthor     = "name the next agent (not the author)"
	lockoutFixOwnerDiffers   = "must differ from Author"
	lockoutIndependentReview = "run independent review of the last specialist change"
)

func TestInitRendersReviewLockout(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}

	read := func(rel string) string {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(got)
	}

	squad := read(".opencode/agents/squad.md")
	for _, want := range []string{lockoutSpawnSpecialists, lockoutNoPatchRejected} {
		if !strings.Contains(squad, want) {
			t.Errorf("squad.md missing %q", want)
		}
	}
	if !strings.Contains(squad, "protocol + files") {
		t.Error("squad.md should document protocol + files, not kernel enforcement")
	}

	for _, rel := range []string{".opencode/agents/tester.md", ".opencode/agents/lead.md"} {
		body := read(rel)
		for _, want := range []string{lockoutRejectInHandoff, lockoutNextNotAuthor} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", rel, want)
			}
		}
	}

	handoff := read(".opencode/skills/squad-handoff/SKILL.md")
	for _, want := range []string{
		"## Review",
		"**Verdict:**",
		"**Author:**",
		"**Fix owner:**",
		"**Reasons:**",
		lockoutFixOwnerDiffers,
	} {
		if !strings.Contains(handoff, want) {
			t.Errorf("squad-handoff missing %q", want)
		}
	}

	review := read(".opencode/commands/squad-review.md")
	if !strings.Contains(strings.ToLower(review), lockoutIndependentReview) {
		t.Errorf("squad-review.md missing %q", lockoutIndependentReview)
	}
}

func TestRecastKeepsReviewerLockout(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := Recast(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".opencode/agents/tester.md", ".opencode/agents/lead.md"} {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		body := string(got)
		if !strings.Contains(body, lockoutRejectInHandoff) || !strings.Contains(body, lockoutNextNotAuthor) {
			t.Errorf("recast %s lost lockout sentences", rel)
		}
	}
}
