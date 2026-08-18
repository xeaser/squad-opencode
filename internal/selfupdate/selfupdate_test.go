package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xeaser/squad-opencode/internal/version"
)

func TestAssetName(t *testing.T) {
	v := version.Version
	cases := []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "squad-oc_" + v + "_windows_amd64.zip"},
		{"linux", "amd64", "squad-oc_" + v + "_linux_amd64.tar.gz"},
		{"linux", "arm64", "squad-oc_" + v + "_linux_arm64.tar.gz"},
		{"darwin", "amd64", "squad-oc_" + v + "_darwin_amd64.tar.gz"},
		{"darwin", "arm64", "squad-oc_" + v + "_darwin_arm64.tar.gz"},
	}
	for _, tc := range cases {
		got := AssetName(tc.goos, tc.goarch)
		if got != tc.want {
			t.Errorf("AssetName(%s, %s) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "squad-oc")
	downloaded := filepath.Join(dir, "downloaded")
	if err := os.WriteFile(current, []byte("old-bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(downloaded, []byte("new-bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceExecutable(current, downloaded); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-bin" {
		t.Fatalf("current = %q, want new-bin", got)
	}
}

func TestReplaceExecutableRenameFailsLeavesNew(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "squad-oc.exe")
	downloaded := filepath.Join(dir, "downloaded")
	if err := os.WriteFile(current, []byte("old-bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(downloaded, []byte("new-bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	renameFile = func(oldpath, newpath string) error {
		return fmt.Errorf("sharing violation")
	}
	t.Cleanup(func() { renameFile = os.Rename })

	err := ReplaceExecutable(current, downloaded)
	if !errors.Is(err, ErrReplacedOnNextStart) {
		t.Fatalf("err = %v, want ErrReplacedOnNextStart", err)
	}
	got, err := os.ReadFile(current + ".new")
	if err != nil {
		t.Fatalf(".new missing: %v", err)
	}
	if string(got) != "new-bin" {
		t.Fatalf(".new = %q, want new-bin", got)
	}
	still, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(still) != "old-bin" {
		t.Fatalf("current changed to %q while locked", still)
	}
}

func TestUpgradeSelfAlready(t *testing.T) {
	srv := releaseServer(t, version.Version, nil)
	msg, err := UpgradeSelf(srv.Client(), version.Repo, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	want := "already " + strings.TrimPrefix(version.Version, "v")
	if msg != want {
		t.Fatalf("msg = %q, want %q", msg, want)
	}
}

func TestUpgradeSelfZip(t *testing.T) {
	payload := []byte("zip-binary-v0.3.0")
	archive := buildZip(t, "squad-oc.exe", payload)
	runUpgradeSelf(t, "windows", "amd64", "squad-oc.exe", archive, payload)
}

func TestUpgradeSelfTarGz(t *testing.T) {
	payload := []byte("targz-binary-v0.3.0")
	archive := buildTarGz(t, "squad-oc", payload)
	runUpgradeSelf(t, "linux", "amd64", "squad-oc", archive, payload)
}

func TestUpgradeSelfRejectsArchiveWithoutBinary(t *testing.T) {
	archive := buildZip(t, "README.txt", []byte("nope"))
	latest := "0.3.0"
	name := assetName(latest, "windows", "amd64")
	srv := releaseServer(t, latest, map[string][]byte{name: archive})

	prevOS, prevArch := currentGOOS, currentGOARCH
	currentGOOS, currentGOARCH = "windows", "amd64"
	t.Cleanup(func() { currentGOOS, currentGOARCH = prevOS, prevArch })

	dummy := filepath.Join(t.TempDir(), "squad-oc.exe")
	if err := os.WriteFile(dummy, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	lookupExe = func() (string, error) { return dummy, nil }
	t.Cleanup(func() { lookupExe = os.Executable })

	_, err := UpgradeSelf(srv.Client(), version.Repo, "0.2.1")
	if err == nil {
		t.Fatal("expected error for archive without squad-oc")
	}
	if !strings.Contains(err.Error(), "squad-oc") {
		t.Fatalf("error should mention missing binary: %v", err)
	}
}

func runUpgradeSelf(t *testing.T, goos, goarch, binName string, archive, payload []byte) {
	t.Helper()
	latest := "0.3.0"
	name := assetName(latest, goos, goarch)
	srv := releaseServer(t, latest, map[string][]byte{name: archive})

	prevOS, prevArch := currentGOOS, currentGOARCH
	currentGOOS, currentGOARCH = goos, goarch
	t.Cleanup(func() { currentGOOS, currentGOARCH = prevOS, prevArch })

	dummy := filepath.Join(t.TempDir(), binName)
	if err := os.WriteFile(dummy, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	lookupExe = func() (string, error) { return dummy, nil }
	t.Cleanup(func() { lookupExe = os.Executable })

	msg, err := UpgradeSelf(srv.Client(), version.Repo, "0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if msg != "updated 0.2.1 → 0.3.0" {
		t.Fatalf("msg = %q", msg)
	}
	got, err := os.ReadFile(dummy)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("replaced binary = %q, want %q", got, payload)
	}
}

func releaseServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			type asset struct {
				Name string `json:"name"`
				URL  string `json:"browser_download_url"`
			}
			payload := struct {
				Tag    string  `json:"tag_name"`
				Assets []asset `json:"assets"`
			}{Tag: tag}
			for name := range assets {
				payload.Assets = append(payload.Assets, asset{
					Name: name,
					URL:  srv.URL + "/download/" + name,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/download/") {
			name := strings.TrimPrefix(r.URL.Path, "/download/")
			body, ok := assets[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(body)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = prev })
	return srv
}

func buildZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
