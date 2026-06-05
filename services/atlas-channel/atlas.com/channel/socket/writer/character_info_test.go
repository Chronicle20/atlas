package writer

import (
	"testing"

	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/monsterbook"

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
	c := character.NewModelBuilder().
		SetId(1).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(col, nil)).
		MustBuild()

	enc := CharacterInfoBody(c, guild.Model{}, nil)
	out := charcb.CharacterInfo{}
	ctx := pt.CreateContext("GMS", 83, 1)
	pt.RoundTrip(t, ctx, enc, out.Decode, nil)

	if out.MonsterBookCover() != 100100 {
		t.Errorf("Character-Info cover = %d, want 100100 (mob id, NOT card id 2380000)", out.MonsterBookCover())
	}
}
