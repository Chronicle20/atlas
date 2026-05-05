package handler

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
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
