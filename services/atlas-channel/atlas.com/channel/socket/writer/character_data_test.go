package writer

import (
	"testing"

	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestBuildCharacterData_MonsterBook(t *testing.T) {
	cards := []monsterbook.Card{}
	c := character.NewModelBuilder().
		SetId(99).
		SetSp("0").
		SetCoverCardId(item.Id(2380001)).
		SetMonsterBookCards(cards).
		MustBuild()

	cd := BuildCharacterData(c, buddylist.Model{})

	if cd.MonsterBook.CoverCardId != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", cd.MonsterBook.CoverCardId)
	}
	if len(cd.MonsterBook.Cards) != len(cards) {
		t.Errorf("card count = %d, want %d", len(cd.MonsterBook.Cards), len(cards))
	}
}
