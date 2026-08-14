package updatecheck

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/version"
)

func TestCheckNoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	prev := APILatest
	APILatest = srv.URL
	defer func() { APILatest = prev }()
	res, err := Check(srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if res.Latest != "" {
		t.Fatal(res)
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
}

func TestCheckUpdateAvailable(t *testing.T) {
	res := checkTag(t, `{"tag_name":"v99.0.0"}`)
	if !strings.Contains(res.Message, "update available") {
		t.Fatalf("want update available, got %q", res.Message)
	}
	if strings.Contains(res.Message, "up to date") {
		t.Fatalf("newer tag should not say up to date: %q", res.Message)
	}
}

func TestCheckSameVersionWithoutVPrefix(t *testing.T) {
	res := checkTag(t, fmt.Sprintf(`{"tag_name":"%s"}`, version.Version))
	if !strings.Contains(res.Message, "up to date") {
		t.Fatalf("want up to date for unprefixed tag, got %q", res.Message)
	}
}

func checkTag(t *testing.T, body string) Result {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prev := APILatest
	APILatest = srv.URL
	t.Cleanup(func() { APILatest = prev })
	res, err := Check(srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return res
}
