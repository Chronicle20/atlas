package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/monsterbook"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestBuildCharacterData_MonsterBook(t *testing.T) {
	cards := []monsterbook.Card{}
	col, err := monsterbook.Extract(monsterbook.CollectionRestModel{CoverCardId: item.Id(2380001)})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	c := character.NewModelBuilder().
		SetId(99).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(col, cards)).
		MustBuild()

	cd := BuildCharacterData(c, buddylist.Model{}, _map.Id(0))

	if cd.MonsterBook.CoverCardId != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", cd.MonsterBook.CoverCardId)
	}
	if len(cd.MonsterBook.Cards) != len(cards) {
		t.Errorf("card count = %d, want %d", len(cd.MonsterBook.Cards), len(cards))
	}
}
