package cmd

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

func TestFnamedocOrderCoversVersionKeys(t *testing.T) {
	in := map[string]bool{}
	for _, v := range fnamedocOrder {
		in[v] = true
	}
	for _, k := range matrix.VersionKeys {
		if !in[k] {
			t.Errorf("VersionKeys entry %q missing from fnamedoc order slice", k)
		}
	}
}
