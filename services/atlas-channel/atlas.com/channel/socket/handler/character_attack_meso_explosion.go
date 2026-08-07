package handler

import (
	"atlas-channel/drop"
)

// validateMesoExplosion checks the exploded-drop list of a Meso Explosion
// attack against the drops in the attacker's field (task-150 FR-5/FR-6/FR-7):
// the listed count must not exceed the skill's attackCount, ids must be
// unique, every id must exist in the field, and every drop must be a meso
// drop (Meso() > 0 — same predicate as the pickup consumer). The field-scoped
// fieldDrops map structurally enforces the same-field/instance check.
//
// Returns (offendingDropId, false) when the attack must be rejected — 0 when
// the failure is the over-max count, which has no single offending drop — and
// (0, true) when valid. An empty list validates trivially: the player can
// swing with nothing to detonate.
func validateMesoExplosion(dropIds []uint32, fieldDrops map[uint32]drop.Model, maxCount uint32) (uint32, bool) {
	if uint32(len(dropIds)) > maxCount {
		return 0, false
	}
	seen := make(map[uint32]struct{}, len(dropIds))
	for _, id := range dropIds {
		if _, dup := seen[id]; dup {
			return id, false
		}
		seen[id] = struct{}{}
		d, ok := fieldDrops[id]
		if !ok {
			return id, false
		}
		if d.Meso() == 0 {
			return id, false
		}
	}
	return 0, true
}
