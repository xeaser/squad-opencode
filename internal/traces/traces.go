// Package traces stores local JSONL spans and exports OTLP JSON.
package traces

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/squad"
)

// Span is a local recorded interval.
type Span struct {
	Name       string            `json:"name"`
	TraceID    string            `json:"traceId"`
	SpanID     string            `json:"spanId"`
	Start      time.Time         `json:"start"`
	End        time.Time         `json:"end"`
	Status     string            `json:"status"` // OK | ERROR
	Attributes map[string]string `json:"attributes"`
}

// Dir is ResolveDir + "traces".
func Dir(projectRoot string) string {
	return filepath.Join(squad.ResolveDir(projectRoot), "traces")
}

func spansPath(projectRoot string) string {
	return filepath.Join(Dir(projectRoot), "spans.jsonl")
}

// Append writes one JSONL span to traces/spans.jsonl.
func Append(projectRoot string, s Span) error {
	if projectRoot == "" {
		return fmt.Errorf("project root is required")
	}
	if s.TraceID == "" {
		id, err := newHex(16)
		if err != nil {
			return err
		}
		s.TraceID = id
	}
	if s.SpanID == "" {
		id, err := newHex(8)
		if err != nil {
			return err
		}
		s.SpanID = id
	}
	if s.Status == "" {
		s.Status = "OK"
	}
	if s.Attributes == nil {
		s.Attributes = map[string]string{}
	}
	if err := os.MkdirAll(Dir(projectRoot), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(spansPath(projectRoot), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(s)
}

// List returns the last n spans (file order). last <= 0 returns all.
func List(projectRoot string, last int) ([]Span, error) {
	path := spansPath(projectRoot)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var spans []Span
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var s Span
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if last > 0 && len(spans) > last {
		spans = spans[len(spans)-last:]
	}
	return spans, nil
}

// FormatTable renders a human-readable table. Empty input is "(no traces)".
func FormatTable(spans []Span) string {
	if len(spans) == 0 {
		return "(no traces)\n"
	}
	var b strings.Builder
	b.WriteString("NAME\tSTATUS\tSTART\tDURATION\tATTRIBUTES\n")
	for _, s := range spans {
		dur := s.End.Sub(s.Start)
		if dur < 0 {
			dur = 0
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n",
			s.Name, s.Status, s.Start.UTC().Format(time.RFC3339), dur.Round(time.Millisecond), formatAttrs(s.Attributes))
	}
	return b.String()
}

func formatAttrs(attrs map[string]string) string {
	if len(attrs) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+attrs[k])
	}
	return strings.Join(parts, " ")
}

// ExportOTLPFile writes an ExportTraceServiceRequest-shaped OTLP JSON file.
func ExportOTLPFile(spans []Span, dest string) error {
	if dest == "" {
		return fmt.Errorf("export destination is required")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	payload := otlpExport{
		ResourceSpans: []otlpResourceSpans{{
			Resource: otlpResource{
				Attributes: []otlpKeyValue{stringAttr("service.name", "squad-oc")},
			},
			ScopeSpans: []otlpScopeSpans{{
				Scope: otlpScope{Name: "squad-oc"},
				Spans: toOTLPSpans(spans),
			}},
		}},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(dest, data, 0o644)
}

func toOTLPSpans(spans []Span) []otlpSpan {
	out := make([]otlpSpan, 0, len(spans))
	for _, s := range spans {
		code := 1
		if strings.EqualFold(s.Status, "ERROR") {
			code = 2
		}
		attrs := make([]otlpKeyValue, 0, len(s.Attributes))
		keys := make([]string, 0, len(s.Attributes))
		for k := range s.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			attrs = append(attrs, stringAttr(k, s.Attributes[k]))
		}
		out = append(out, otlpSpan{
			TraceID:           s.TraceID,
			SpanID:            s.SpanID,
			Name:              s.Name,
			Kind:              1,
			StartTimeUnixNano: unixNanoString(s.Start),
			EndTimeUnixNano:   unixNanoString(s.End),
			Attributes:        attrs,
			Status:            otlpStatus{Code: code},
		})
	}
	return out
}

func stringAttr(key, value string) otlpKeyValue {
	return otlpKeyValue{Key: key, Value: otlpValue{StringValue: value}}
}

func unixNanoString(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixNano())
}

func newHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type otlpExport struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource     `json:"resource"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

type otlpScopeSpans struct {
	Scope otlpScope  `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpScope struct {
	Name string `json:"name"`
}

type otlpSpan struct {
	TraceID           string         `json:"traceId"`
	SpanID            string         `json:"spanId"`
	Name              string         `json:"name"`
	Kind              int            `json:"kind"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	EndTimeUnixNano   string         `json:"endTimeUnixNano"`
	Attributes        []otlpKeyValue `json:"attributes"`
	Status            otlpStatus     `json:"status"`
}

type otlpStatus struct {
	Code int `json:"code"`
}

type otlpKeyValue struct {
	Key   string    `json:"key"`
	Value otlpValue `json:"value"`
}

type otlpValue struct {
	StringValue string `json:"stringValue"`
}
