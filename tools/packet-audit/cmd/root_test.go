package cmd

import (
	"bytes"
	"testing"
)

func TestRootHelp(t *testing.T) {
	var buf bytes.Buffer
	rc := Run([]string{"--help"}, &buf)
	if rc != 0 {
		t.Fatalf("--help exit=%d, want 0", rc)
	}
	if !bytes.Contains(buf.Bytes(), []byte("packet-audit")) {
		t.Fatalf("--help output missing tool name: %q", buf.String())
	}
}
