package traces

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xeaser/squad-opencode/internal/squad"
)

func sampleSpan(name string, start time.Time, attrs map[string]string) Span {
	return Span{
		Name:       name,
		TraceID:    "aabbccddeeff00112233445566778899",
		SpanID:     "1122334455667788",
		Start:      start,
		End:        start.Add(1500 * time.Millisecond),
		Status:     "OK",
		Attributes: attrs,
	}
}

func TestAppendListExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	want := sampleSpan("squad-oc.run", start, map[string]string{
		"agent":        "squad",
		"prompt_bytes": "4",
	})
	if err := Append(root, want); err != nil {
		t.Fatal(err)
	}
	got, err := List(root, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	s := got[0]
	if s.Name != want.Name || s.TraceID != want.TraceID || s.SpanID != want.SpanID || s.Status != "OK" {
		t.Fatalf("%+v", s)
	}
	if !s.Start.Equal(want.Start) || !s.End.Equal(want.End) {
		t.Fatalf("times %v %v", s.Start, s.End)
	}
	if s.Attributes["agent"] != "squad" || s.Attributes["prompt_bytes"] != "4" {
		t.Fatalf("attrs %+v", s.Attributes)
	}

	path := filepath.Join(Dir(root), "spans.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"name":"squad-oc.run"`) {
		t.Fatalf("jsonl: %s", raw)
	}

	dest := filepath.Join(root, "otlp.json")
	if err := ExportOTLPFile(got, dest); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	rs, ok := payload["resourceSpans"].([]any)
	if !ok || len(rs) != 1 {
		t.Fatalf("resourceSpans: %s", body)
	}
	first, _ := rs[0].(map[string]any)
	resource, _ := first["resource"].(map[string]any)
	if !jsonHasAttr(resource["attributes"], "service.name", "squad-oc") {
		t.Fatalf("resource: %s", body)
	}
	scopes, _ := first["scopeSpans"].([]any)
	if len(scopes) != 1 {
		t.Fatalf("scopeSpans: %s", body)
	}
	scope0, _ := scopes[0].(map[string]any)
	scope, _ := scope0["scope"].(map[string]any)
	if scope["name"] != "squad-oc" {
		t.Fatalf("scope: %s", body)
	}
	spans, _ := scope0["spans"].([]any)
	if len(spans) != 1 {
		t.Fatalf("spans: %s", body)
	}
	sp, _ := spans[0].(map[string]any)
	if sp["traceId"] != want.TraceID || sp["spanId"] != want.SpanID || sp["name"] != want.Name {
		t.Fatalf("otlp span: %s", body)
	}
	if sp["startTimeUnixNano"] != unixNanoString(want.Start) {
		t.Fatalf("start nano: %v want %s", sp["startTimeUnixNano"], unixNanoString(want.Start))
	}
	if sp["endTimeUnixNano"] != unixNanoString(want.End) {
		t.Fatalf("end nano: %v want %s", sp["endTimeUnixNano"], unixNanoString(want.End))
	}
	if !jsonHasAttr(sp["attributes"], "agent", "squad") {
		t.Fatalf("span attrs: %s", body)
	}
	status, _ := sp["status"].(map[string]any)
	if statusCode(status["code"]) != 1 {
		t.Fatalf("status: %+v", status)
	}
}

func TestListLastNAndMissing(t *testing.T) {
	root := t.TempDir()
	got, err := List(root, 20)
	if err != nil || len(got) != 0 {
		t.Fatalf("missing file: %+v %v", got, err)
	}
	base := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	for i, name := range []string{"one", "two", "three"} {
		s := sampleSpan(name, base.Add(time.Duration(i)*time.Second), nil)
		s.TraceID = ""
		s.SpanID = ""
		if err := Append(root, s); err != nil {
			t.Fatal(err)
		}
	}
	got, err = List(root, 2)
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if got[0].Name != "two" || got[1].Name != "three" {
		t.Fatalf("want last two, got %+v", got)
	}
	if got[0].TraceID == "" || got[0].SpanID == "" {
		t.Fatal("Append should fill ids")
	}
	all, err := List(root, 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("last<=0 should return all: %d %v", len(all), err)
	}
}

func TestDirUsesResolveDir(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(squad.ResolveDir(root), "traces")
	if Dir(root) != want {
		t.Fatalf("got %s want %s", Dir(root), want)
	}
}

func TestFormatTable(t *testing.T) {
	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	out := FormatTable([]Span{sampleSpan("squad-oc.run", start, map[string]string{"agent": "squad"})})
	if !strings.Contains(out, "squad-oc.run") || !strings.Contains(out, "OK") || !strings.Contains(out, "agent=squad") {
		t.Fatal(out)
	}
	if FormatTable(nil) != "(no traces)\n" {
		t.Fatalf("empty: %q", FormatTable(nil))
	}
}

func TestInitGitignoreIgnoresTraces(t *testing.T) {
	root := t.TempDir()
	if _, err := squad.WriteDefaultPreset(squad.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".squad", ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "traces/") {
		t.Fatalf(".squad/.gitignore missing traces/: %s", data)
	}
}

func jsonHasAttr(raw any, key, value string) bool {
	arr, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if m["key"] != key {
			continue
		}
		val, _ := m["value"].(map[string]any)
		return val["stringValue"] == value
	}
	return false
}

func statusCode(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return -1
	}
}
