package share

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/squad-opencode/squad-opencode/internal/squad"
)

func TestUpstreamAndPack(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	pack := t.TempDir()
	agentDir := filepath.Join(pack, ".opencode", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "extra.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddUpstream(root, "local", pack); err != nil {
		t.Fatal(err)
	}
	list, err := ListUpstreams(root)
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
	n, err := SyncUpstream(root, "local")
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "extra.md")); err != nil {
		t.Fatal(err)
	}
	n, err = InstallPack(root, pack)
	if err != nil || n < 1 {
		t.Fatal(n, err)
	}
	if err := RemoveUpstream(root, "local"); err != nil {
		t.Fatal(err)
	}
	list, _ = ListUpstreams(root)
	if len(list) != 0 {
		t.Fatal(list)
	}
}

func TestLink(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err := Link(root, other); err != nil {
		t.Fatal(err)
	}
	if squad.Detect(root).Config.LinkPath == "" {
		t.Fatal("expected link")
	}
}
