package heal

import (
	"sync"

	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// selectRecipients prepends the caster to the in-range party members
// returned by the shared resolver. Caller is responsible for the
// shared resolver call so test stubs don't need to fake the whole
// processor stack.
func selectRecipients(caster recipient, party []handler.PartyRecipient) []recipient {
	out := make([]recipient, 0, 1+len(party))
	out = append(out, caster)
	for _, p := range party {
		out = append(out, recipient{
			Id:    p.Id,
			X:     p.X,
			Y:     p.Y,
			Hp:    p.Hp,
			MaxHp: p.MaxHp,
		})
	}
	return out
}

// warnedRectangles dedupes the missing-LT/RB warning per (skillId,
// skillLevel) tuple per-process. Reset between tests via
// resetWarnedRectangles.
var warnedRectangles sync.Map // key: uint64 (skillId<<8 | level)

func warnIfMissingRectangle(skillId skill2.Id, skillLevel byte, e effect.Model, logf func()) {
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

// resetWarnedRectangles is exposed for tests.
func resetWarnedRectangles() {
	warnedRectangles = sync.Map{}
}
