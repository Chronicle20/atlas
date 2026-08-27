package writer

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/equipment"
	eqslot "atlas-channel/equipment/slot"
	"atlas-channel/guild"
	"atlas-channel/ring"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	slot2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ringsDoc renders a JSON:API "rings" list document for one ACTIVE couple
// half belonging to characterId, mirroring ring/processor_test.go's fixture
// shape (task-269 task 9).
func ringsDoc(characterId uint32, cashId int64) string {
	return fmt.Sprintf(
		`{"data":[{"id":"%s","type":"rings","attributes":{"pairId":"%s","characterId":%d,"partnerCharacterId":200,"assetId":1,"itemTemplateId":1112001,"ringType":"COUPLE","state":"ACTIVE","cashId":%d,"partnerCashId":2222,"partnerName":"Partner"}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`,
		uuid.New(), uuid.New(), characterId, cashId,
	)
}

// seedRingCache populates ctx's tenant-scoped ring cache for characterId with
// one ACTIVE couple half via the real Populate REST path (an httptest.Server
// standing in for atlas-cashshop), so the encoder tests below exercise the
// same cache Task 10 built rather than reaching into ring package internals.
// Task 12 (character-load population) is not wired yet, so tests must seed
// the cache directly like this.
func seedRingCache(t *testing.T, ctx context.Context, characterId uint32, cashId int64) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprint(w, ringsDoc(characterId, cashId))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/api/")

	p := ring.NewProcessor(logrus.New(), ctx)
	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate: %v", err)
	}
}

func equippedRing1(cashId int64) func(c character.Model) character.Model {
	return func(c character.Model) character.Model {
		eq := equipment.NewModel()
		a := asset.NewBuilderWithId(1, uuid.New(), 1112001).SetCashId(cashId).MustBuild()
		eq.Set(slot2.Type("ring1"), eqslot.Model{Position: -12, CashEquipable: &a})
		return character.CloneModel(c).SetEquipment(eq).MustBuild()
	}
}

// TestCharacterSpawnBodyCarriesRings pins Task 11's wiring of the ring
// processor into CharacterSpawnBody (task-269 site: socket write
// character_spawn.go:60, replacing the Task-7 packetmodel.RingSet{} zero
// value).
func TestCharacterSpawnBodyCarriesRings(t *testing.T) {
	t.Run("ring present", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 83, 1)
		const characterId = uint32(100)
		const cashId = int64(1111)
		seedRingCache(t, ctx, characterId, cashId)

		c := character.NewBuilder().SetId(characterId).SetSp("0").MustBuild()
		c = equippedRing1(cashId)(c)

		withRing := CharacterSpawnBody(c, nil, guild.Model{}, true)
		gotWithRing := pt.Encode(t, ctx, withRing, nil)

		// Same character, same equipped ring1 -- the only difference from
		// gotWithRing is that this character id's ring cache was never
		// seeded, so GetRingSet returns an empty RingSet. Isolates the
		// RingSet contribution from the avatar's cash-equip visual bytes.
		cUncached := character.NewBuilder().SetId(characterId + 1).SetSp("0").MustBuild()
		cUncached = equippedRing1(cashId)(cUncached)
		empty := CharacterSpawnBody(cUncached, nil, guild.Model{}, true)
		gotEmpty := pt.Encode(t, ctx, empty, nil)

		// One populated PairRing arm writes byte(1) + int64 + int64 + uint32 =
		// 21 bytes (model.PairRing.EncodeField, libs/atlas-packet/model/ring.go)
		// in place of the empty arm's byte(0) = 1 byte -- a difference of 20
		// bytes, with everything else in the two bodies identical.
		wantDelta := 20
		if gotDelta := len(gotWithRing) - len(gotEmpty); gotDelta != wantDelta {
			t.Fatalf("length delta = %d, want %d (with=%d, empty=%d)", gotDelta, wantDelta, len(gotWithRing), len(gotEmpty))
		}
	})

	t.Run("no ring", func(t *testing.T) {
		// FR-9 guard (task-13 addendum A2): compare against a captured
		// baseline byte string, not a second live encode of the same input.
		// An encode-vs-encode comparison is a tautology -- it cannot catch a
		// regression that changes the empty-ring path, because both sides
		// would change together.
		ctx := pt.CreateContext("GMS", 83, 1)
		const characterId = uint32(200)

		c := character.NewBuilder().SetId(characterId).SetSp("0").MustBuild()
		enc := CharacterSpawnBody(c, nil, guild.Model{}, true)
		got := pt.Encode(t, ctx, enc, nil)

		want, err := hex.DecodeString("c8000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000ffff000000000000000000000000000000000000000000000000000000000000d6ff0600000000010000000000000000000000000000000000000000")
		if err != nil {
			t.Fatalf("decode fixture: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("cache-miss encode changed:\n got %x\nwant %x", got, want)
		}
	})
}
