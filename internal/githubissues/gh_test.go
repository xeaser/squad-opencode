package githubissues

import "testing"

func TestParseListJSON(t *testing.T) {
	raw := []byte(`[{"number":1,"title":"Hi","state":"OPEN"}]`)
	issues, err := ParseListJSON(raw)
	if err != nil || len(issues) != 1 || issues[0].Number != 1 {
		t.Fatalf("%v %+v", err, issues)
	}
}
