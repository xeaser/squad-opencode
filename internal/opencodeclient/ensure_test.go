package opencodeclient

import (
	"context"
	"errors"
	"testing"
)

func TestIsDefaultLocal(t *testing.T) {
	yes := []string{
		DefaultBaseURL,
		"http://127.0.0.1:4096/",
		"http://localhost:4096",
		"http://localhost:4096/",
		"127.0.0.1:4096",
	}
	for _, s := range yes {
		if !IsDefaultLocal(s) {
			t.Fatalf("want default local: %s", s)
		}
	}
	no := []string{
		"http://127.0.0.1:5000",
		"http://localhost:4097",
		"http://192.168.1.9:4096",
		"http://devbox:4096",
		"https://opencode.example.com",
		"http://0.0.0.0:4096",
	}
	for _, s := range no {
		if IsDefaultLocal(s) {
			t.Fatalf("must not auto-start: %s", s)
		}
	}
}

func TestResolveBaseURL(t *testing.T) {
	t.Setenv(EnvBaseURL, "")
	if got := ResolveBaseURL(""); got != DefaultBaseURL {
		t.Fatal(got)
	}
	t.Setenv(EnvBaseURL, "http://devbox:4096")
	if got := ResolveBaseURL(""); got != "http://devbox:4096" {
		t.Fatal(got)
	}
	if got := ResolveBaseURL("http://127.0.0.1:5000"); got != "http://127.0.0.1:5000" {
		t.Fatal(got)
	}
}

func TestEnsureAPIAttaches(t *testing.T) {
	prevP, prevS := ProbeFn, StartFn
	t.Cleanup(func() { ProbeFn, StartFn = prevP, prevS })
	started := 0
	ProbeFn = func(context.Context, string) ProbeResult {
		return ProbeResult{Reachable: true, BaseURL: DefaultBaseURL}
	}
	StartFn = func(string) error {
		started++
		return nil
	}
	res, err := EnsureAPI(context.Background(), "", t.TempDir())
	if err != nil || !res.Attached || res.Started || started != 0 {
		t.Fatalf("%+v %v started=%d", res, err, started)
	}
	if res.Message != "attached to "+DefaultBaseURL {
		t.Fatal(res.Message)
	}
}

func TestEnsureAPIStartsDefaultLocal(t *testing.T) {
	prevP, prevS := ProbeFn, StartFn
	t.Cleanup(func() { ProbeFn, StartFn = prevP, prevS })
	probes := 0
	ProbeFn = func(context.Context, string) ProbeResult {
		probes++
		if probes == 1 {
			return ProbeResult{Reachable: false}
		}
		return ProbeResult{Reachable: true}
	}
	StartFn = func(string) error { return nil }
	res, err := EnsureAPI(context.Background(), DefaultBaseURL, t.TempDir())
	if err != nil || !res.Started || res.Attached {
		t.Fatalf("%+v %v", res, err)
	}
	if res.Message != "started opencode serve at "+DefaultBaseURL {
		t.Fatal(res.Message)
	}
}

func TestEnsureAPIDoesNotStartCustomURL(t *testing.T) {
	prevP, prevS := ProbeFn, StartFn
	t.Cleanup(func() { ProbeFn, StartFn = prevP, prevS })
	started := 0
	ProbeFn = func(context.Context, string) ProbeResult {
		return ProbeResult{Reachable: false}
	}
	StartFn = func(string) error {
		started++
		return nil
	}
	_, err := EnsureAPI(context.Background(), "http://127.0.0.1:5000", t.TempDir())
	if err == nil || started != 0 {
		t.Fatalf("err=%v started=%d", err, started)
	}
	t.Setenv(EnvBaseURL, "http://devbox:4096")
	_, err = EnsureAPI(context.Background(), "", t.TempDir())
	if err == nil || started != 0 {
		t.Fatalf("env err=%v started=%d", err, started)
	}
}

func TestEnsureAPIStartError(t *testing.T) {
	prevP, prevS := ProbeFn, StartFn
	t.Cleanup(func() { ProbeFn, StartFn = prevP, prevS })
	ProbeFn = func(context.Context, string) ProbeResult {
		return ProbeResult{Reachable: false}
	}
	StartFn = func(string) error { return errors.New("no binary") }
	_, err := EnsureAPI(context.Background(), "", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
}
