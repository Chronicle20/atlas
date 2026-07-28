package handler

import (
	"atlas-channel/data/skill/effect"
	"sync"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// warnedRectangles dedupes the missing-LT/RB warning per (skillId,
// skillLevel) tuple per-process. Reset between tests via
// ResetWarnedRectangles.
var warnedRectangles sync.Map // key: uint64 (skillId<<8 | level)

// WarnIfMissingRectangle logs once per (skillId, skillLevel) when the effect
// has no LT/RB rectangle. Shared by handlers whose recipient resolution
// depends on the rect-based party selector (heal, timeleap, ...).
func WarnIfMissingRectangle(skillId skill2.Id, skillLevel byte, e effect.Model, logf func()) {
	lt, rb := e.LT(), e.RB()
	if lt.X() != 0 || lt.Y() != 0 || rb.X() != 0 || rb.Y() != 0 {
		return
	}
	key := uint64(skillId)<<8 | uint64(skillLevel)
	if _, loaded := warnedRectangles.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	logf()
}

// ResetWarnedRectangles is exposed for tests.
func ResetWarnedRectangles() {
	warnedRectangles = sync.Map{}
}
