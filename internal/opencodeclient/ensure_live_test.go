package opencodeclient

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestLiveEnsureAPI(t *testing.T) {
	if os.Getenv("SQUAD_OC_LIVE") == "" {
		t.Skip("set SQUAD_OC_LIVE=1 to probe a real opencode serve")
	}
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	// Serve may outlive the test; use the dummy sandbox, not t.TempDir()
	// (Windows cannot delete a cwd that still holds opencode serve).
	dir := `D:\xAI\squad-oc-dummy`
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := EnsureAPI(ctx, "", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Attached && !res.Started {
		t.Fatalf("%+v", res)
	}
	if !ProbeServer(ctx, res.BaseURL).Reachable {
		t.Fatal("server not reachable after ensure")
	}
	t.Log(res.Message)
}
