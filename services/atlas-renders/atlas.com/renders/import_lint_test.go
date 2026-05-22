package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoForbiddenWzImports is the load-bearing CI gate guaranteeing
// atlas-renders never accidentally pulls in the WZ subpackages that belong
// to the ingest-side path only.
//
// Following the lazy-map-render refactor (docs/tasks/task-071-.../
// lazy-map-render.md), atlas-renders DOES import wz/canvas/mapimage so
// it can lazily composite map layers at render time from Map.wz. The
// constraint that survives is narrower: the atlas-packer + deterministic
// PNG encoder (used only by the canonical character-atlas baseline path
// in ingest) and the icons pipeline (used only by ingest workers to emit
// per-entity icon PNGs) remain off-limits.
func TestNoForbiddenWzImports(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./...").Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	forbidden := []string{
		"github.com/Chronicle20/atlas/libs/atlas-wz/atlas",
		"github.com/Chronicle20/atlas/libs/atlas-wz/atlas/pngenc",
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
