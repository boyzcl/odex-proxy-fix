package common

import "testing"

func TestMergeNoProxy(t *testing.T) {
	got := MergeNoProxy("example.com,localhost", "localhost,127.0.0.1")
	want := "example.com,localhost,127.0.0.1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
