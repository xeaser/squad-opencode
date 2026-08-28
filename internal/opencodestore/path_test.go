package opencodestore

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func TestResolveDBPathDefaultXDGThenHome(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "/home/u", nil }
	t.Cleanup(func() { userHomeDir = orig })

	p, expl, err := ResolveDBPath("", nil, func(string) string { return "" })
	if err != nil || expl {
		t.Fatalf("default: %q %v %v", p, expl, err)
	}
	want := filepath.Join("/home/u", ".local", "share", "opencode", "opencode.db")
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}

	p, expl, err = ResolveDBPath("", nil, func(k string) string {
		if k == "XDG_DATA_HOME" {
			return "/xdg/data"
		}
		return ""
	})
	if err != nil || expl {
		t.Fatal(err)
	}
	want = filepath.Join("/xdg/data", "opencode", "opencode.db")
	if p != want {
		t.Fatalf("xdg: %q want %q", p, want)
	}
}

func TestResolveDBPathEnvAndConfig(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "/home/u", nil }
	t.Cleanup(func() { userHomeDir = orig })

	abs := filepath.Join(t.TempDir(), "custom.db")
	p, expl, err := ResolveDBPath("/proj", nil, func(k string) string {
		if k == "OPENCODE_DB" {
			return abs
		}
		return ""
	})
	if err != nil || !expl || p != filepath.Clean(abs) {
		t.Fatalf("abs env: %q %v %v", p, expl, err)
	}

	p, expl, err = ResolveDBPath("/proj", nil, func(k string) string {
		if k == "OPENCODE_DB" {
			return "chan.db"
		}
		return ""
	})
	if err != nil || !expl {
		t.Fatal(err)
	}
	wantRel := filepath.Join("/home/u", ".local", "share", "opencode", "chan.db")
	if p != wantRel {
		t.Fatalf("rel env: %q want %q", p, wantRel)
	}

	cfg := &squad.Config{OpenCodeDB: "local.db"}
	p, expl, err = ResolveDBPath("/proj", cfg, func(string) string { return "" })
	if err != nil || !expl || p != filepath.Join("/proj", "local.db") {
		t.Fatalf("config: %q %v %v", p, expl, err)
	}

	p, expl, err = ResolveDBPath("/proj", cfg, func(k string) string {
		if k == "OPENCODE_DB" {
			return abs
		}
		return ""
	})
	if err != nil || !expl || p != filepath.Clean(abs) {
		t.Fatalf("env wins: %q %v %v", p, expl, err)
	}

	_, _, err = ResolveDBPath("", nil, func(k string) string {
		if k == "OPENCODE_DB" {
			return ":memory:"
		}
		return ""
	})
	if err == nil || !strings.Contains(err.Error(), ":memory:") {
		t.Fatalf("memory: %v", err)
	}
}
