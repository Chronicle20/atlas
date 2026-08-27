package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/equipslot"
	"atlas-channel/character/teleportrock"
	"atlas-channel/monsterbook"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestBuildCharacterData_MonsterBook(t *testing.T) {
	cards := []monsterbook.Card{}
	col, err := monsterbook.Extract(monsterbook.CollectionRestModel{CoverCardId: item.Id(2380001)})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	c := character.NewBuilder().
		SetId(99).
		SetSp("0").
		SetMonsterBook(monsterbook.NewModel(col, cards)).
		MustBuild()

	cd := BuildCharacterData(logrus.New(), context.Background(), c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

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
	c := character.NewBuilder().
		SetId(99).
		SetSp("0").
		MustBuild()
	trm := teleportrock.NewModel([]_map.Id{100000000}, []_map.Id{104040000, 220000000})
	cd := BuildCharacterData(logrus.New(), context.Background(), c, buddylist.Model{}, _map.Id(0), trm)
	if len(cd.TeleportMaps) != 1 || cd.TeleportMaps[0] != 100000000 {
		t.Fatalf("teleport maps: %v", cd.TeleportMaps)
	}
	if len(cd.VipTeleportMaps) != 2 {
		t.Fatalf("vip maps: %v", cd.VipTeleportMaps)
	}
}

// TestBuildCharacterData_SpawnPoint pins both halves of the task-272 fix: the
// model value reaches the wire struct, and the uint32 -> byte narrowing at the
// wire boundary truncates rather than erroring. Truncation above 255 is a
// pre-existing property of the wire format (one byte), asserted here so it is
// documented rather than latent.
func TestBuildCharacterData_SpawnPoint(t *testing.T) {
	tests := []struct {
		name string
		set  uint32
		want byte
	}{
		{name: "in range", set: 7, want: 7},
		{name: "truncates above 255", set: 256, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := character.NewBuilder().
				SetId(99).
				SetSp("0").
				SetSpawnPoint(tt.set).
				MustBuild()

			cd := BuildCharacterData(logrus.New(), context.Background(), c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

			if cd.Stats.SpawnPoint != tt.want {
				t.Errorf("Stats.SpawnPoint = %d, want %d", cd.Stats.SpawnPoint, tt.want)
			}
		})
	}
}

// TestEquipSlotExtExpireFor_NoActiveExtension pins R3: a character with no
// active equip-slot extension keeps the ZeroTime sentinel, not an error.
func TestEquipSlotExtExpireFor_NoActiveExtension(t *testing.T) {
	if got := equipSlotExtExpireFor(nil); got != ZeroTime {
		t.Errorf("equipSlotExtExpireFor(nil) = %d, want ZeroTime %d", got, ZeroTime)
	}
	if got := equipSlotExtExpireFor([]equipslot.RestModel{}); got != ZeroTime {
		t.Errorf("equipSlotExtExpireFor([]) = %d, want ZeroTime %d", got, ZeroTime)
	}
}

// TestEquipSlotExtExpireFor_ActiveExtension pins R4's derived FILETIME
// conversion against a hand-computed value for a fixed UTC instant --
// 2024-01-01T00:00:00Z, Unix 1704067200 -- so a wrong conversion (e.g. a
// units mismatch or the wrong epoch constant) fails loudly instead of
// silently rendering a wrong expiry on the client.
func TestEquipSlotExtExpireFor_ActiveExtension(t *testing.T) {
	expiresAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exts := []equipslot.RestModel{{CharacterId: 99, SlotIndex: -59, ExpiresAt: expiresAt}}

	got := equipSlotExtExpireFor(exts)

	const want int64 = 133485408000000000 // 1704067200 * 10_000_000 + 116444736000000000
	if got != want {
		t.Fatalf("equipSlotExtExpireFor = %d, want %d", got, want)
	}
	if got == ZeroTime {
		t.Fatalf("equipSlotExtExpireFor regressed to the ZeroTime sentinel for an active extension")
	}
}
