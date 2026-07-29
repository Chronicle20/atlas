package handler

import (
	"atlas-channel/data/skill/effect"
	"testing"
)

func TestWarnIfMissingRectangle_OncePerTuple(t *testing.T) {
	defer ResetWarnedRectangles()

	calls := 0
	logf := func() { calls++ }

	WarnIfMissingRectangle(2301002, 1, effect.Model{}, logf)
	WarnIfMissingRectangle(2301002, 1, effect.Model{}, logf)
	WarnIfMissingRectangle(2301002, 2, effect.Model{}, logf)

	if calls != 2 {
		t.Fatalf("warn calls = %d, want 2 (one per (id, level))", calls)
	}
}
