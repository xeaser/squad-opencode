package opencodeclient

import (
	"context"
	"testing"
)

func TestFakeRunner(t *testing.T) {
	f := &FakeRunner{Text: "hello"}
	res, err := f.Run(context.Background(), RunRequest{Prompt: "p", Agent: "squad"})
	if err != nil || res.Text != "hello" || len(f.Calls) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
}
