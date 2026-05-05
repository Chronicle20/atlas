package handler

import (
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// calculateBoundingBox derives the (x1, y1, x2, y2) target rectangle for a
// monster-buff skill cast. Mirrors Cosmic StatEffect.calculateBoundingBox.
//
// When the caster faces left, the rectangle is (casterPos + lt) → (casterPos + rb).
// When the caster faces right, the rectangle mirrors about the caster's X:
// x1 = casterX - rb.X, x2 = casterX - lt.X. The y bounds are always
// (casterY + lt.Y) → (casterY + rb.Y).
//
// The returned tuple is not normalized — atlas-monsters' GetInFieldRect
// normalizes (min, max) on its side, so callers can pass either ordering.
func calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16) {
	if facingLeft {
		x1 = casterX + int16(lt.X())
		y1 = casterY + int16(lt.Y())
		x2 = casterX + int16(rb.X())
		y2 = casterY + int16(rb.Y())
	} else {
		x1 = casterX - int16(rb.X())
		y1 = casterY + int16(lt.Y())
		x2 = casterX - int16(lt.X())
		y2 = casterY + int16(rb.Y())
	}
	return
}

// hasEffectBbox reports whether the effect carries a non-degenerate target
// rectangle. The WZ "no rect contract" sentinel is all four components zero;
// any non-zero component (even a single int) indicates the effect prescribes
// a rect. No v83 skill ships a literal zero-area effect, so the conflation is
// safe in production.
func hasEffectBbox(lt, rb point.Model) bool {
	return lt.X() != 0 || lt.Y() != 0 || rb.X() != 0 || rb.Y() != 0
}

// intersectMobIds partitions client mob ids into "applied" (also present in
// server) and "anomaly" (client-only) lists. Server-only ids are dropped per
// FR-4.1: the client's omission is treated as authoritative for "did not
// target". Result preserves client order (FR-4.4) so wire traces remain
// readable. Both returned slices are nil if the corresponding bucket is
// empty (callers checking len() observe the same behavior either way).
func intersectMobIds(client, server []uint32) (applied, anomaly []uint32) {
	if len(client) == 0 {
		return nil, nil
	}
	serverSet := make(map[uint32]struct{}, len(server))
	for _, id := range server {
		serverSet[id] = struct{}{}
	}
	for _, id := range client {
		if _, ok := serverSet[id]; ok {
			applied = append(applied, id)
		} else {
			anomaly = append(anomaly, id)
		}
	}
	return applied, anomaly
}

// mobBuffApplyKind returns the reflect-kind that gates a mob-affecting buff
// apply (FR-4.6). Today only Priest Doom is in `isMobAffectingBuff` for the
// apply branch; future apply-style status skills are added here as they are
// wired in. Returning "" tells the orchestrator to skip the reflect check
// entirely and emit a debug "unclassified kind" log — the cast still proceeds.
//
// Crash/Dispel kinds continue to come from dispelSkillClass (common.go) and
// are not handled here.
func mobBuffApplyKind(skillId skill2.Id) string {
	switch {
	case skill2.Is(skillId, skill2.PriestDoomId):
		return monster2.ReflectKindMagical
	default:
		return ""
	}
}
