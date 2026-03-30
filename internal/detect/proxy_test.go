package detect

import "testing"

func TestParseProxyURL(t *testing.T) {
	host, port, ok := parseProxyURL("http://127.0.0.1:7897")
	if !ok || host != "127.0.0.1" || port != 7897 {
		t.Fatalf("unexpected parse result: %v %v %v", host, port, ok)
	}
}
