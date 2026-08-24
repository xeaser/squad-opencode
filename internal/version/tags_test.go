package version

import "testing"

func TestIsStableTag(t *testing.T) {
	if !IsStableTag("v0.5.1") {
		t.Fatal("v0.5.1")
	}
	if IsStableTag("v0.5.1-rc.1") || IsStableTag("v0.0.0-guard-test") || IsStableTag("0.5.1") {
		t.Fatal("prerelease or unprefixed")
	}
}

func TestPreviousStableTag(t *testing.T) {
	tags := []string{"v0.4.0", "v0.5.0", "v0.5.1", "v0.5.2", "v0.0.0-guard-test", "v0.5.2-rc.1"}
	if got := PreviousStableTag("v0.5.2", tags); got != "v0.5.1" {
		t.Fatalf("got %q want v0.5.1", got)
	}
	if got := PreviousStableTag("v0.5.0", tags); got != "v0.4.0" {
		t.Fatalf("got %q want v0.4.0", got)
	}
	if got := PreviousStableTag("v0.4.0", tags); got != "" {
		t.Fatalf("first stable previous = %q", got)
	}
	if got := PreviousStableTag("v0.5.1-rc.1", tags); got != "" {
		t.Fatalf("prerelease current = %q", got)
	}
	if got := PreviousStableTag("v0.10.0", []string{"v0.9.0", "v0.10.0"}); got != "v0.9.0" {
		t.Fatalf("numeric sort: %q", got)
	}
}
