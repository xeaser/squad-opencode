package updatecheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/version"
)

func TestCheckNoReleases(t *testing.T) {
	isolateCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	prev := APILatest
	APILatest = srv.URL
	defer func() { APILatest = prev }()
	res, err := Check(srv.Client(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Latest != "" {
		t.Fatal(res)
	}
	if res.Status != "unknown" {
		t.Fatalf("status: %q", res.Status)
	}
}

func TestCheckUpToDate(t *testing.T) {
	res := checkTag(t, fmt.Sprintf(`{"tag_name":"v%s"}`, version.Version))
	if !strings.Contains(res.Message, "up to date") {
		t.Fatalf("want up to date, got %q", res.Message)
	}
	if strings.Contains(res.Message, "update available") {
		t.Fatalf("same version should not say update available: %q", res.Message)
	}
	if res.Status != "up to date" {
		t.Fatalf("status: %q", res.Status)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	res := checkTag(t, `{"tag_name":"v99.0.0"}`)
	if !strings.Contains(res.Message, "update available") {
		t.Fatalf("want update available, got %q", res.Message)
	}
	if strings.Contains(res.Message, "up to date") {
		t.Fatalf("newer tag should not say up to date: %q", res.Message)
	}
	if res.Status != "update available" {
		t.Fatalf("status: %q", res.Status)
	}
}

func TestCheckSameVersionWithoutVPrefix(t *testing.T) {
	res := checkTag(t, fmt.Sprintf(`{"tag_name":"%s"}`, version.Version))
	if !strings.Contains(res.Message, "up to date") {
		t.Fatalf("want up to date for unprefixed tag, got %q", res.Message)
	}
	if res.Status != "up to date" {
		t.Fatalf("status: %q", res.Status)
	}
}

func TestCheckUsesFreshCache(t *testing.T) {
	isolateCache(t)
	cached := Result{
		Local:   version.Version,
		Latest:  "v88.0.0",
		Status:  "update available",
		Message: "update available — local " + version.Version + ", latest v88.0.0",
	}
	seedCache(t, cached, time.Now())

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0"}`))
	}))
	t.Cleanup(srv.Close)
	prev := APILatest
	APILatest = srv.URL
	t.Cleanup(func() { APILatest = prev })

	res, err := Check(srv.Client(), false)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 0 {
		t.Fatalf("expected cache hit (0 server calls), got %d", hits)
	}
	if res.Latest != "v88.0.0" {
		t.Fatalf("want cached latest v88.0.0, got %q", res.Latest)
	}
}

func TestCheckRefreshBypassesCache(t *testing.T) {
	isolateCache(t)
	seedCache(t, Result{
		Local:   version.Version,
		Latest:  "v88.0.0",
		Status:  "update available",
		Message: "cached",
	}, time.Now())

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0"}`))
	}))
	t.Cleanup(srv.Close)
	prev := APILatest
	APILatest = srv.URL
	t.Cleanup(func() { APILatest = prev })

	res, err := Check(srv.Client(), true)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 server call with refresh, got %d", hits)
	}
	if res.Latest != "v99.0.0" {
		t.Fatalf("want latest v99.0.0, got %q", res.Latest)
	}
}

func TestResultJSONIncludesStatus(t *testing.T) {
	res := Result{
		Local:   "0.1.0",
		Latest:  "v0.2.0",
		Status:  "update available",
		Message: formatCompare("0.1.0", "v0.2.0"),
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["status"] != "update available" {
		t.Fatalf("json status: %q body=%s", m["status"], b)
	}
	if m["local"] != "0.1.0" || m["latest"] != "v0.2.0" {
		t.Fatalf("json fields: %s", b)
	}
	if !strings.Contains(m["message"], "update available") {
		t.Fatalf("message: %q", m["message"])
	}
	if !strings.Contains(formatCompare("0.1.0", "v0.2.0"), "update available") {
		t.Fatal("formatCompare")
	}
}

func checkTag(t *testing.T, body string) Result {
	t.Helper()
	isolateCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prev := APILatest
	APILatest = srv.URL
	t.Cleanup(func() { APILatest = prev })
	res, err := Check(srv.Client(), true)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func isolateCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	// macOS uses HOME/Library/Caches
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Ensure cache parent exists for path construction checks
	_ = CachePath()
}

func seedCache(t *testing.T, res Result, checkedAt time.Time) {
	t.Helper()
	path := CachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := struct {
		CheckedAt string `json:"checkedAt"`
		Result    Result `json:"result"`
	}{
		CheckedAt: checkedAt.UTC().Format(time.RFC3339),
		Result:    res,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
