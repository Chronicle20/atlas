package writer

import (
	"testing"

	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/teleportrock"
	"atlas-channel/monsterbook"

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

	cd := BuildCharacterData(c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

	if cd.MonsterBook.CoverCardId != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", cd.MonsterBook.CoverCardId)
	}
	if len(cd.MonsterBook.Cards) != len(cards) {
		t.Errorf("card count = %d, want %d", len(cd.MonsterBook.Cards), len(cards))
	}
}

func TestBuildCharacterData_TeleportMaps(t *testing.T) {
	// Bare character.Model{} panics in RemainingSp() (parses the Sp string);
	// reuse the same builder as TestBuildCharacterData_MonsterBook.
	c := character.NewModelBuilder().
		SetId(99).
		SetSp("0").
		MustBuild()
	trm := teleportrock.NewModel([]_map.Id{100000000}, []_map.Id{104040000, 220000000})
	cd := BuildCharacterData(c, buddylist.Model{}, _map.Id(0), trm)
	if len(cd.TeleportMaps) != 1 || cd.TeleportMaps[0] != 100000000 {
		t.Fatalf("teleport maps: %v", cd.TeleportMaps)
	}
	if len(cd.VipTeleportMaps) != 2 {
		t.Fatalf("vip maps: %v", cd.VipTeleportMaps)
	}
}
