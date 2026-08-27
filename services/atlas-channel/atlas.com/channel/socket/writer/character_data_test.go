package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/equipslot"
	"atlas-channel/character/teleportrock"
	"atlas-channel/monsterbook"
	"atlas-channel/ring"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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

	cd := BuildCharacterData(logrus.New(), pt.CreateContext("GMS", 83, 1), c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

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
	cd := BuildCharacterData(logrus.New(), pt.CreateContext("GMS", 83, 1), c, buddylist.Model{}, _map.Id(0), trm)
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

			cd := BuildCharacterData(logrus.New(), pt.CreateContext("GMS", 83, 1), c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

			if cd.Stats.SpawnPoint != tt.want {
				t.Errorf("Stats.SpawnPoint = %d, want %d", cd.Stats.SpawnPoint, tt.want)
			}
		})
	}
}

// TestBuildCharacterData_Rings covers the RECORD block call site
// (cd.Rings = ring.NewProcessor(l, ctx).GetRingRecords(c.Id())) at the
// writer level -- carried from Task 11's review (non-blocking). Only
// ring.GetRingRecords was covered in isolation before; nothing seeded the
// ring cache and asserted on cd.Rings itself, so a swap of this call site
// for GetRingSet's AVATAR-block shape would have passed the whole suite.
// The seeded half's PartnerName must survive into cd.Rings.Couple.
func TestBuildCharacterData_Rings(t *testing.T) {
	const characterId = uint32(99)
	const cashId = int64(1111)
	const partnerCashId = int64(2222)
	const partnerCharacterId = uint32(200)
	const partnerName = "Partner"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":[{"id":"%s","type":"rings","attributes":{"pairId":"%s","characterId":%d,"partnerCharacterId":%d,"assetId":1,"itemTemplateId":1112001,"ringType":"COUPLE","state":"ACTIVE","cashId":%d,"partnerCashId":%d,"partnerName":%q}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`,
			uuid.New(), uuid.New(), characterId, partnerCharacterId, cashId, partnerCashId, partnerName)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	ctx := pt.CreateContext("GMS", 83, 1)
	if err := ring.NewProcessor(logrus.New(), ctx).Populate(characterId); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	c := character.NewBuilder().
		SetId(characterId).
		SetSp("0").
		MustBuild()

	cd := BuildCharacterData(logrus.New(), ctx, c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})

	if len(cd.Rings.Couple) != 1 {
		t.Fatalf("cd.Rings.Couple = %+v, want exactly 1 seeded half", cd.Rings.Couple)
	}
	got := cd.Rings.Couple[0]
	if got.PairCharacterName != partnerName {
		t.Errorf("PairCharacterName = %q, want the seeded partnerName %q", got.PairCharacterName, partnerName)
	}
	if got.PairCharacterId != partnerCharacterId {
		t.Errorf("PairCharacterId = %d, want %d", got.PairCharacterId, partnerCharacterId)
	}
	if got.OwnSN != cashId || got.PairSN != partnerCashId {
		t.Errorf("OwnSN/PairSN = %d/%d, want %d/%d", got.OwnSN, got.PairSN, cashId, partnerCashId)
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
