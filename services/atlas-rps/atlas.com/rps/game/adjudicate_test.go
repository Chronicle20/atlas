package game_test

import (
	"testing"

	"atlas-rps/game"
)

func TestAdjudicateAllCombinations(t *testing.T) {
	cases := []struct {
		p, o game.Throw
		want game.Outcome
	}{
		{game.ThrowRock, game.ThrowScissors, game.OutcomeWin},
		{game.ThrowRock, game.ThrowPaper, game.OutcomeLose},
		{game.ThrowRock, game.ThrowRock, game.OutcomeTie},
		{game.ThrowPaper, game.ThrowRock, game.OutcomeWin},
		{game.ThrowPaper, game.ThrowScissors, game.OutcomeLose},
		{game.ThrowPaper, game.ThrowPaper, game.OutcomeTie},
		{game.ThrowScissors, game.ThrowPaper, game.OutcomeWin},
		{game.ThrowScissors, game.ThrowRock, game.OutcomeLose},
		{game.ThrowScissors, game.ThrowScissors, game.OutcomeTie},
	}
	for _, c := range cases {
		if got := game.Adjudicate(c.p, c.o); got != c.want {
			t.Errorf("Adjudicate(%v,%v)=%v want %v", c.p, c.o, got, c.want)
		}
	}
}

func TestDeterministicThrowSource(t *testing.T) {
	fixed := game.ThrowSource(func() game.Throw {
		return game.ThrowPaper
	})
	if got := fixed(); got != game.ThrowPaper {
		t.Errorf("fixed ThrowSource() = %v, want %v", got, game.ThrowPaper)
	}
}
