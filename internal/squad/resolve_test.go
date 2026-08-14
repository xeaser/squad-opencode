package squad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDirFollowsLink(t *testing.T) {
	local := t.TempDir()
	shared := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: local, ProjectDescription: "local-app"}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: shared, ProjectDescription: "shared-team"}); err != nil {
		t.Fatal(err)
	}
	if ResolveDir(local) != SquadDir(local) {
		t.Fatal("expected local before link")
	}

	// Link via the other project's root (contains .squad/).
	path, err := ResolveLinkTarget(shared)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetLink(local, path); err != nil {
		t.Fatal(err)
	}
	if ResolveDir(local) != SquadDir(shared) {
		t.Fatalf("got %s want %s", ResolveDir(local), SquadDir(shared))
	}
	members, err := ReadTeam(local)
	if err != nil {
		t.Fatal(err)
	}
	team, _ := os.ReadFile(filepath.Join(ResolveDir(local), "team.md"))
	if !strings.Contains(string(team), "shared-team") {
		t.Fatalf("linked team not used: %s", team)
	}
	if len(members) < 4 {
		t.Fatal(members)
	}

	if err := ClearLink(local); err != nil {
		t.Fatal(err)
	}
	team, _ = os.ReadFile(filepath.Join(ResolveDir(local), "team.md"))
	if !strings.Contains(string(team), "local-app") {
		t.Fatalf("unlink should restore local: %s", team)
	}
}

func TestResolveLinkTargetAcceptsSquadDir(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveLinkTarget(SquadDir(root))
	if err != nil || got != SquadDir(root) {
		t.Fatalf("%s %v", got, err)
	}
	got, err = ResolveLinkTarget(root)
	if err != nil || got != SquadDir(root) {
		t.Fatalf("%s %v", got, err)
	}
}

func TestLinkRejectedWhenExternalized(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: other}); err != nil {
		t.Fatal(err)
	}
	if _, err := ExternalizeTo(root, filepath.Join(t.TempDir(), "ext")); err != nil {
		t.Fatal(err)
	}
	if err := SetLink(root, SquadDir(other)); err == nil {
		t.Fatal("expected error")
	}
}

func TestExternalizeRejectedWhenLinked(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultPreset(InitOptions{ProjectRoot: other}); err != nil {
		t.Fatal(err)
	}
	if err := SetLink(root, SquadDir(other)); err != nil {
		t.Fatal(err)
	}
	if _, err := ExternalizeTo(root, filepath.Join(t.TempDir(), "ext")); err == nil {
		t.Fatal("expected error")
	}
}
