package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusReport(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{
		ProjectRoot:        root,
		Preset:             "default",
		ProjectDescription: "status test",
	}); err != nil {
		t.Fatal(err)
	}

	decPath := filepath.Join(ResolveDir(root), "decisions.md")
	decBody := `# Decisions

Append new decisions at the top (newest first).

### 2026-08-14 — Ship richer status

- **Status:** accepted
- **Context:** Task 4
- **Decision:** Extract StatusReport
- **Consequences:** Testable status output

`
	if err := os.WriteFile(decPath, []byte(decBody), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Host:") {
		t.Fatalf("missing Host:\n%s", out)
	}
	if !strings.Contains(out, "Lead") {
		t.Fatalf("missing member name:\n%s", out)
	}
	if !strings.Contains(out, "Decisions") {
		t.Fatalf("missing Decisions section:\n%s", out)
	}
	if !strings.Contains(out, "Ship richer status") {
		t.Fatalf("missing decisions excerpt:\n%s", out)
	}
	// Title line alone should not be the only decisions content; excerpt skips # title.
	if strings.Contains(out, "# Decisions\n") {
		t.Fatalf("decisions excerpt should skip # title:\n%s", out)
	}
	// Watch section absent without ralph-status.json
	if strings.Contains(out, "Watch") {
		t.Fatalf("unexpected Watch section:\n%s", out)
	}
}

func TestStatusReportWatchSection(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{
		ProjectRoot: root,
		Preset:      "default",
	}); err != nil {
		t.Fatal(err)
	}
	statusPath := filepath.Join(ResolveDir(root), "ralph-status.json")
	body := `{"lastPoll":"2026-08-14T12:00:00Z","lastSummary":"polled 2 issues"}`
	if err := os.WriteFile(statusPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Watch") {
		t.Fatalf("missing Watch section:\n%s", out)
	}
	if !strings.Contains(out, "2026-08-14T12:00:00Z") {
		t.Fatalf("missing lastPoll:\n%s", out)
	}
	if !strings.Contains(out, "polled 2 issues") {
		t.Fatalf("missing lastSummary:\n%s", out)
	}
}

func TestStatusReportNotInitialized(t *testing.T) {
	root := t.TempDir()
	_, err := StatusReport(root)
	if err == nil {
		t.Fatal("expected error when not initialized")
	}
}

func TestStatusReportRemoteLink(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: other, ProjectDescription: "shared"}); err != nil {
		t.Fatal(err)
	}
	if err := SetRemoteLink(root, SquadDir(other), "https://example.com/team.git", "main", "abcdef1234567890"); err != nil {
		t.Fatal(err)
	}
	got, err := StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "https://example.com/team.git") || !strings.Contains(got, "abcdef1") {
		t.Fatal(got)
	}
}

func TestStatusReportRemoteLinkSHAWithoutRef(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: other, ProjectDescription: "shared"}); err != nil {
		t.Fatal(err)
	}
	if err := SetRemoteLink(root, SquadDir(other), "https://example.com/team.git", "", "abcdef1234567890"); err != nil {
		t.Fatal(err)
	}
	got, err := StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Link rev: abcdef1") {
		t.Fatalf("expected SHA when LinkRef is empty:\n%s", got)
	}
}

func TestStatusReportModels(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	out, err := StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Model: (session)") {
		t.Fatalf("empty orchestrator:\n%s", out)
	}
	if err := SetSquadModel(root, "xai/grok-3"); err != nil {
		t.Fatal(err)
	}
	if err := SetMemberModel(root, "Lead", "anthropic/claude-sonnet-4-5"); err != nil {
		t.Fatal(err)
	}
	out, err = StatusReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Model: xai/grok-3") {
		t.Fatalf("orchestrator:\n%s", out)
	}
	if !strings.Contains(out, "anthropic/claude-sonnet-4-5") {
		t.Fatalf("override:\n%s", out)
	}
	if !strings.Contains(out, "xai/grok-3  (team)") && !strings.Contains(out, "xai/grok-3 (team)") {
		t.Fatalf("inherit marker:\n%s", out)
	}
}
