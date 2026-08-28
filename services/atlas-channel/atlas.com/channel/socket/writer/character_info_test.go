package writer

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/monsterbook"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterInfoBody_CoverIsMobId(t *testing.T) {
	col, err := monsterbook.Extract(monsterbook.CollectionRestModel{
		CoverCardId:    item.Id(2380000),
		CoverMonsterId: 100100,
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	c := character.NewBuilder().
		SetId(1).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(col, nil)).
		MustBuild()

	enc := CharacterInfoBody(c, guild.Model{}, nil, charcb.MountInfo{})
	out := charcb.CharacterInfo{}
	ctx := pt.CreateContext("GMS", 83, 1)
	pt.RoundTrip(t, ctx, enc, out.Decode, nil)

	if out.MonsterBookCover() != 100100 {
		t.Errorf("Character-Info cover = %d, want 100100 (mob id, NOT card id 2380000)", out.MonsterBookCover())
	}
}

// TestCharacterInfoBodySetsMarriageFlag pins Task 11's wiring of the ring
// processor's Marriage arm into CharacterInfoBody's hasMarriageRing flag
// (character_info.go, replacing the Task-7 hardcoded false). Marriage-ring
// acquisition is a PRD non-goal (ring.Processor.GetRingSet's doc comment):
// the Marriage arm is always nil, so no marriage half cached is the only
// reachable state, and the flag byte must always encode 0x00.
func TestCharacterInfoBodySetsMarriageFlag(t *testing.T) {
	c := character.NewBuilder().
		SetId(1).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(monsterbook.Collection{}, nil)).
		MustBuild()

	enc := CharacterInfoBody(c, guild.Model{}, nil, charcb.MountInfo{})
	out := charcb.CharacterInfo{}
	ctx := pt.CreateContext("GMS", 83, 1)
	pt.RoundTrip(t, ctx, enc, out.Decode, nil)

	if out.HasMarriageRing() {
		t.Errorf("HasMarriageRing() = true, want false (Marriage arm is always nil)")
	}
}
