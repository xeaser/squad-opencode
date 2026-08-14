package githubissues

import (
	"strings"
	"testing"
)

func TestParseListJSON(t *testing.T) {
	raw := []byte(`[{"number":1,"title":"Hi","state":"OPEN"}]`)
	issues, err := ParseListJSON(raw)
	if err != nil || len(issues) != 1 || issues[0].Number != 1 {
		t.Fatalf("%v %+v", err, issues)
	}
}

func TestGHListerArgsLabel(t *testing.T) {
	g := GHLister{Labels: []string{"bug"}}
	got := strings.Join(g.args(), " ")
	if !strings.Contains(got, "--label bug") {
		t.Fatalf("want --label bug in %q", got)
	}
	if !strings.Contains(got, "--limit 20") {
		t.Fatalf("default limit 20 in %q", got)
	}
}
