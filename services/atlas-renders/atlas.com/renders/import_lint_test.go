package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoForbiddenWzImports is the load-bearing CI gate guaranteeing
// atlas-renders never accidentally pulls in the heavy WZ parser, canvas
// renderer, atlas packer, or icon pipeline. Those subpackages live in
// libs/atlas-wz for the extractor's benefit; renders only consumes the
// pure-data sidecars (manifest, maplayout).
func TestNoForbiddenWzImports(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./...").Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	forbidden := []string{
		"github.com/Chronicle20/atlas/libs/atlas-wz/wz",
		"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property",
		"github.com/Chronicle20/atlas/libs/atlas-wz/crypto",
		"github.com/Chronicle20/atlas/libs/atlas-wz/canvas",
		"github.com/Chronicle20/atlas/libs/atlas-wz/atlas",
		"github.com/Chronicle20/atlas/libs/atlas-wz/atlas/pngenc",
		"github.com/Chronicle20/atlas/libs/atlas-wz/mapimage",
		"github.com/Chronicle20/atlas/libs/atlas-wz/icons",
	}
	text := string(out)
	for _, f := range forbidden {
		for _, line := range strings.Split(text, "\n") {
			if line == f {
				t.Errorf("forbidden wz subpackage imported by atlas-renders: %s", f)
			}
		}
	}
}
