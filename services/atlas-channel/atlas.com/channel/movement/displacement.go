package movement

import (
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Displaces reports whether a movement path actually relocates the entity —
// i.e. whether the position the path ends on differs from the position it
// started on.
//
// A serverbound MOVE packet is NOT proof that the player walked. The client
// also flushes a path whose fragments land on the coordinates it started
// from, notably when a new action begins after a period of standing still.
// Callers that want "did this player move?" (as opposed to "did a move
// packet arrive?") must ask this, not merely count packets.
//
// The end position is folded with the SAME folder the movement processor
// uses to derive the authoritative position it publishes (ForCharacter), so
// this can never disagree with where the server believes the character is.
// Fragments that carry no landing coordinates (Jump, StartFallDown) leave
// the position untouched, exactly as they do there.
func Displaces(m model.Movement) bool {
	s, err := model2.Fold(model2.FixedProvider(m.Elements), summaryProvider(m.StartX, m.StartY, 0), folder)()
	if err != nil {
		// The fold's only failure mode is a provider error; treat an
		// undecidable path as movement so the conservative branch wins.
		return true
	}
	return s.X != m.StartX || s.Y != m.StartY
}
