package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeRestoresHostLeavesTeam(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{
		ProjectRoot:        root,
		Preset:             "default",
		ProjectDescription: "Keep me",
	}); err != nil {
		t.Fatal(err)
	}

	teamPath := filepath.Join(root, ".squad", "team.md")
	teamBefore, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	agent := filepath.Join(root, ".opencode", "agents", "squad.md")
	if err := os.WriteFile(agent, []byte("MUTATED\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) == 0 {
		t.Fatalf("expected squad.md updated, got %+v", res)
	}

	got, err := os.ReadFile(agent)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "MUTATED\n" || !strings.Contains(string(got), "You are **Squad**") {
		t.Fatalf("agent not restored from template:\n%s", got)
	}

	teamAfter, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(teamAfter) != string(teamBefore) {
		t.Fatal("team.md must not change on upgrade")
	}
	if !strings.Contains(string(teamAfter), "Keep me") {
		t.Fatal("description lost")
	}
}

func TestUpgradeDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	agent := filepath.Join(root, ".opencode", "agents", "lead.md")
	if err := os.WriteFile(agent, []byte("MUTATED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) == 0 {
		t.Fatal("dry-run should report updates")
	}
	got, _ := os.ReadFile(agent)
	if string(got) != "MUTATED\n" {
		t.Fatal("dry-run must not write")
	}
}

func TestUpgradeRequiresInit(t *testing.T) {
	_, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpgradeUnchangedWhenMatch(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	res, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 0 {
		t.Fatalf("fresh init should be unchanged, updated=%v", res.Updated)
	}
	if len(res.Unchanged) == 0 {
		t.Fatal("expected unchanged host files")
	}
}

func TestUpgradeDoesNotTouchKnowledge(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	know := filepath.Join(root, ".squad", "agents", "backend", "knowledge.md")
	custom := "# Backend knowledge\n\nI learned APIs live in /api.\n"
	if err := os.WriteFile(know, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := UpgradeHostFiles(UpgradeOptions{ProjectRoot: root, Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(know)
	if string(got) != custom {
		t.Fatalf("knowledge overwritten:\n%s", got)
	}
}
