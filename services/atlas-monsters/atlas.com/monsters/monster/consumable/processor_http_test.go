package consumable_test

import (
	"atlas-monsters/monster/consumable"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// consumableDoc renders a JSON:API single-resource "consumables" document,
// mirroring atlas-data's GET /data/consumables/{itemId} response shape
// (rest.go's RestModel).
func consumableDoc(id string, create, monsterId, monsterHp, bridleProp uint32, bridlePropChg float64) string {
	return fmt.Sprintf(
		`{"data":{"id":"%s","type":"consumables","attributes":{"create":%d,"monsterId":%d,"monsterHP":%d,"bridleProp":%d,"bridlePropChg":%v}}}`,
		id, create, monsterId, monsterHp, bridleProp, bridlePropChg,
	)
}

// TestGetById_HTTPRoundTrip exercises GetById's real unmarshal path
// (requests.Provider -> JSON:API decode -> Extract), not an injected seam.
// This proves the RestModel struct tags and Extract actually decode a live
// atlas-data response into the expected Model.
func TestGetById_HTTPRoundTrip(t *testing.T) {
	const itemId = uint32(2270002)
	const wantCreate = uint32(4031868)
	const wantMonsterId = uint32(9300157)
	const wantMonsterHp = uint32(40)
	const wantBridleProp = uint32(50)
	const wantBridlePropChg = 1.2

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/data/consumables/2270002" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(consumableDoc("2270002", wantCreate, wantMonsterId, wantMonsterHp, wantBridleProp, wantBridlePropChg)))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	m, err := consumable.NewProcessor(l, ctx).GetById(itemId)
	if err != nil {
		t.Fatal(err)
	}
	if m.Id() != itemId {
		t.Fatalf("Id() = %d, want %d", m.Id(), itemId)
	}
	if m.Create() != wantCreate {
		t.Fatalf("Create() = %d, want %d (create attribute decode failed)", m.Create(), wantCreate)
	}
	if m.MonsterId() != wantMonsterId {
		t.Fatalf("MonsterId() = %d, want %d (monsterId attribute decode failed)", m.MonsterId(), wantMonsterId)
	}
	if m.MonsterHp() != wantMonsterHp {
		t.Fatalf("MonsterHp() = %d, want %d (monsterHP attribute decode failed)", m.MonsterHp(), wantMonsterHp)
	}
	if m.BridleProp() != wantBridleProp {
		t.Fatalf("BridleProp() = %d, want %d (bridleProp attribute decode failed)", m.BridleProp(), wantBridleProp)
	}
	if m.BridlePropChg() != wantBridlePropChg {
		t.Fatalf("BridlePropChg() = %v, want %v (bridlePropChg attribute decode failed)", m.BridlePropChg(), wantBridlePropChg)
	}
}

// TestGetById_HTTPRoundTrip_NotFound proves a real upstream 404 maps to
// requests.ErrNotFound through GetById.
func TestGetById_HTTPRoundTrip_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	if _, err := consumable.NewProcessor(l, ctx).GetById(404); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("err = %v, want errors.Is(_, requests.ErrNotFound)", err)
	}
}
