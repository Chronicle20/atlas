package registrations

import (
	channelhandler "atlas-channel/skill/handler"
	"testing"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestMonsterMagnetHandlersRegistered guards the blank-import wiring. The
// monstermagnet package registers itself from init(); if nothing imports it the
// handler compiles fine and is simply never installed.
func TestMonsterMagnetHandlersRegistered(t *testing.T) {
	for _, id := range []skill2.Identity{
		skill2.HeroMonsterMagnet,
		skill2.PaladinMonsterMagnet,
		skill2.DarkKnightMonsterMagnet,
	} {
		if _, ok := channelhandler.Lookup(id); !ok {
			t.Fatalf("no Handler registered for identity [%v]", id)
		}
		if _, ok := channelhandler.LookupAttackCast(id); ok {
			t.Fatalf("identity [%v] must NOT be in the AttackCastHandler registry: the magnet arrives on the use-skill opcode and deals no damage", id)
		}
	}
}
