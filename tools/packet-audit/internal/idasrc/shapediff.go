package idasrc

import "fmt"

// ShapeVerdict describes the outcome of comparing a hand-authored wire shape
// against the live-derived shape extracted from IDA.
type ShapeVerdict int

const (
	// ShapeVerified: the live shape matches the hand shape under audit-grade width
	// tolerance (fieldEquivalent). No divergence found.
	ShapeVerified ShapeVerdict = iota

	// ShapeDivergent: a real read-order or width divergence exists in the known
	// (non-Unresolved) portion. This is a genuine audit finding.
	ShapeDivergent

	// ShapeUnverifiable: the live shape contains an Unresolved span (an
	// undecompilable helper), and the known prefix up to that span matches the
	// hand shape. The remainder cannot be validated — not a Divergent finding.
	ShapeUnverifiable
)

// String returns a human-readable name for the verdict.
func (v ShapeVerdict) String() string {
	switch v {
	case ShapeVerified:
		return "verified"
	case ShapeDivergent:
		return "divergent"
	case ShapeUnverifiable:
		return "unverifiable"
	}
	return "unknown"
}

// ValidateShape compares a hand-authored wire shape against the live-derived
// shape using audit-grade width tolerance (fieldEquivalent). A live Unresolved
// read marks an undecompilable span: the shape is Unverifiable (not Divergent)
// provided the prefix up to the first Unresolved matches; a divergence in the
// known prefix is still a Divergent finding regardless.
//
// Returns the verdict and a detail string. The detail string is non-empty only
// when the verdict is ShapeDivergent.
func ValidateShape(hand, live []FieldCall) (ShapeVerdict, string) {
	// Find the first Unresolved index in live (-1 if none).
	unresolvedAt := -1
	for i, c := range live {
		if c.Op == Unresolved {
			unresolvedAt = i
			break
		}
	}

	if unresolvedAt == -1 {
		// No Unresolved in live: the full shapes must match.
		if len(hand) != len(live) {
			return ShapeDivergent, fmt.Sprintf("length: hand %d vs live %d", len(hand), len(live))
		}
		for i := range hand {
			if !fieldEquivalent(hand[i].Op, live[i].Op) {
				return ShapeDivergent, fmt.Sprintf("at [%d]: hand=%s live=%s", i, hand[i].Op, live[i].Op)
			}
		}
		return ShapeVerified, ""
	}

	// There is an Unresolved at index unresolvedAt.
	// Compare the known prefix: live[0:unresolvedAt] vs hand[0:unresolvedAt].
	p := unresolvedAt

	// If hand is shorter than the known prefix, it's a length divergence.
	if len(hand) < p {
		return ShapeDivergent, fmt.Sprintf("hand has only %d field(s) but live known prefix requires %d", len(hand), p)
	}

	// Check per-position equivalence within the known prefix.
	for i := 0; i < p; i++ {
		if !fieldEquivalent(hand[i].Op, live[i].Op) {
			return ShapeDivergent, fmt.Sprintf("at [%d]: hand=%s live=%s (before Unresolved at [%d])", i, hand[i].Op, live[i].Op, p)
		}
	}

	// Known prefix matches; the Unresolved span makes the rest unverifiable.
	beyond := len(hand) - p
	return ShapeUnverifiable, fmt.Sprintf("Unresolved span at [%d]; %d hand field(s) beyond verified prefix", p, beyond)
}
