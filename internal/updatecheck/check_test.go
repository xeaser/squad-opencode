package updatecheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestResultFormat(t *testing.T) {
	r := Result{Local: "0.2.0", Latest: "v0.3.0", Message: "ok"}
	if r.Latest != "v0.3.0" {
		t.Fatal(r)
	}
}
