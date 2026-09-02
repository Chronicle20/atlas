package main

import (
	"bytes"
	"fmt"
)

// overlaySuffixes maps each overlay to the suffix its topic names carry.
//
//	-PLACEHOLDER_ATLAS_ENV          -- overlays/pr, isolated mode: this
//	  environment gets its own topics, suffixed with its own env token.
//	-PLACEHOLDER_BASELINE_ENVIRONMENT -- overlays/pr-sparse, sparse mode:
//	  this environment shares the BASELINE's topics, so it must name them
//	  the way the baseline names them. Note this is not the same as "no
//	  suffix": the baseline overlay suffixes every topic with its own
//	  environment id (`-main` today), so an unsuffixed name addresses a
//	  topic nobody publishes to. That was the atlas-login crash-loop of
//	  2026-08-20 -- see docs/tasks/task-232-sparse-ephemeral-environments/
//	  bug-sparse-baseline-scoping.md.
var overlaySuffixes = map[string]string{
	"main":      "-main",
	"pr":        "-PLACEHOLDER_ATLAS_ENV",
	"pr-sparse": "-PLACEHOLDER_BASELINE_ENVIRONMENT",
}

// EmitOverlayBlock renders m as the `      - TOKEN=TOKEN<suffix>` literals
// Splice inserts into an overlay kustomization.yaml's
// `# BEGIN generated:topics` / `# END generated:topics` region, six-space
// indented to match the surrounding configMapGenerator literals sequence.
func (m Manifest) EmitOverlayBlock(suffix string) []byte {
	var buf bytes.Buffer
	for _, e := range m.Topics {
		fmt.Fprintf(&buf, "      - %s=%s%s\n", e.Token, e.Token, suffix)
	}
	return buf.Bytes()
}

// EmitComposeBlock renders m as deploy/compose/.env.example's topic lines:
// shell `TOKEN=TOKEN` assignments, manifest order, identity mapping, no
// indent and no quoting.
func (m Manifest) EmitComposeBlock() []byte {
	var buf bytes.Buffer
	for _, e := range m.Topics {
		fmt.Fprintf(&buf, "%s=%s\n", e.Token, e.Token)
	}
	return buf.Bytes()
}
