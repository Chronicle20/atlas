package game

import (
	"math/rand"
)

// Outcome represents the result of an RPS round from the player's perspective.
type Outcome int

const (
	OutcomeLose Outcome = iota
	OutcomeTie
	OutcomeWin
)

// beats maps a Throw to the Throw it defeats.
var beats = map[Throw]Throw{
	ThrowRock:     ThrowScissors,
	ThrowScissors: ThrowPaper,
	ThrowPaper:    ThrowRock,
}

// Adjudicate applies the RPS rules and returns the Outcome from the player's
// perspective. It is a pure function: rock beats scissors, scissors beats
// paper, paper beats rock; equal throws tie. Server authority for adjudication
// lives here (FR-2.2).
func Adjudicate(playerThrow, opponentThrow Throw) Outcome {
	if playerThrow == opponentThrow {
		return OutcomeTie
	}
	if beats[playerThrow] == opponentThrow {
		return OutcomeWin
	}
	return OutcomeLose
}

// ThrowSource produces a Throw, typically the opponent's. Injectable for
// deterministic testing.
type ThrowSource func() Throw

// DefaultThrowSource is the server-authoritative RNG opponent-throw source.
// It uses the top-level math/rand source, which since Go 1.20 is auto-seeded
// and safe for concurrent use by multiple goroutines (atlas-rps calls this
// from concurrent Kafka-consumer goroutines).
func DefaultThrowSource() Throw {
	return Throw(rand.Intn(3))
}
