package playernpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestRestModel_Unmarshal asserts every PRD §5 attribute -- including
// overallJobRank (rest.go's doc comment on RestModel) -- round-trips
// through a JSON:API document decode.
func TestRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "player-npcs",
			"id": "0eaf71e9-2ba7-4443-96bb-886f7dc8213c",
			"attributes": {
				"characterId": 42,
				"name": "Bowman",
				"worldId": 0,
				"mapId": 100000000,
				"scriptId": 9000000,
				"objectId": 7,
				"gender": 1,
				"skin": 0,
				"face": 20000,
				"hair": 30000,
				"jobId": 100,
				"x": 250,
				"cy": 300,
				"fh": 12,
				"rx0": 200,
				"rx1": 300,
				"dir": 1,
				"worldRank": 1,
				"overallRank": 2,
				"worldJobRank": 3,
				"overallJobRank": 4,
				"equipment": [],
				"deployedAt": "2024-01-01T00:00:00Z"
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if rm.GetID() != "0eaf71e9-2ba7-4443-96bb-886f7dc8213c" {
		t.Errorf("GetID() = %q", rm.GetID())
	}
	if rm.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", rm.CharacterId)
	}
	if rm.Name != "Bowman" {
		t.Errorf("Name = %q, want Bowman", rm.Name)
	}
	if rm.WorldId != world.Id(0) {
		t.Errorf("WorldId = %d, want 0", rm.WorldId)
	}
	if rm.MapId != _map.Id(100000000) {
		t.Errorf("MapId = %d, want 100000000", rm.MapId)
	}
	if rm.ScriptId != 9000000 {
		t.Errorf("ScriptId = %d, want 9000000", rm.ScriptId)
	}
	if rm.ObjectId != 7 {
		t.Errorf("ObjectId = %d, want 7", rm.ObjectId)
	}
	if rm.Gender != 1 {
		t.Errorf("Gender = %d, want 1", rm.Gender)
	}
	if rm.Skin != 0 {
		t.Errorf("Skin = %d, want 0", rm.Skin)
	}
	if rm.Face != 20000 {
		t.Errorf("Face = %d, want 20000", rm.Face)
	}
	if rm.Hair != 30000 {
		t.Errorf("Hair = %d, want 30000", rm.Hair)
	}
	if rm.JobId != 100 {
		t.Errorf("JobId = %d, want 100", rm.JobId)
	}
	if rm.X != 250 {
		t.Errorf("X = %d, want 250", rm.X)
	}
	if rm.Cy != 300 {
		t.Errorf("Cy = %d, want 300", rm.Cy)
	}
	if rm.Fh != 12 {
		t.Errorf("Fh = %d, want 12", rm.Fh)
	}
	if rm.Rx0 != 200 {
		t.Errorf("Rx0 = %d, want 200", rm.Rx0)
	}
	if rm.Rx1 != 300 {
		t.Errorf("Rx1 = %d, want 300", rm.Rx1)
	}
	if rm.Dir != 1 {
		t.Errorf("Dir = %d, want 1", rm.Dir)
	}
	if rm.WorldRank != 1 || rm.OverallRank != 2 || rm.WorldJobRank != 3 || rm.OverallJobRank != 4 {
		t.Errorf("ranks = %+v, want {1 2 3 4}", []uint32{rm.WorldRank, rm.OverallRank, rm.WorldJobRank, rm.OverallJobRank})
	}
	wantDeployedAt, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	if !rm.DeployedAt.Equal(wantDeployedAt) {
		t.Errorf("DeployedAt = %v, want %v", rm.DeployedAt, wantDeployedAt)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.CharacterId() != 42 || m.Name() != "Bowman" || m.OverallJobRank() != 4 {
		t.Fatalf("Extract round-trip mismatch: %+v", m)
	}
}

// TestExtract_EquipmentOrder asserts the equipment array decodes into
// (slot, itemId) pairs in the order they appear on the wire.
func TestExtract_EquipmentOrder(t *testing.T) {
	rm := RestModel{
		Id: uuid.New(),
		Equipment: []EquipmentRestModel{
			{Slot: -1, ItemId: 1040002},
			{Slot: -5, ItemId: 1060002},
			{Slot: -101, ItemId: 1322005},
		},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	eq := m.Equipment()
	if len(eq) != 3 {
		t.Fatalf("Equipment len = %d, want 3", len(eq))
	}
	wantSlots := []int16{-1, -5, -101}
	wantItemIds := []uint32{1040002, 1060002, 1322005}
	for i, e := range eq {
		if e.Slot() != wantSlots[i] || e.ItemId() != wantItemIds[i] {
			t.Errorf("Equipment[%d] = (%d, %d), want (%d, %d)", i, e.Slot(), e.ItemId(), wantSlots[i], wantItemIds[i])
		}
	}
}

// TestRestModel_EquipmentRoundTrip asserts a non-empty equipment slice
// round-trips through real jsonapi.Marshal/jsonapi.Unmarshal -- the Task 18
// review left this decode path unexercised (only Extract's own struct
// literal was tested, not the JSON:API decode into EquipmentRestModel).
func TestRestModel_EquipmentRoundTrip(t *testing.T) {
	want := RestModel{
		Id:          uuid.New(),
		CharacterId: 42,
		Name:        "Bowman",
		Equipment: []EquipmentRestModel{
			{Slot: -1, ItemId: 1040002},
			{Slot: -5, ItemId: 1060002},
			{Slot: -101, ItemId: 1322005},
		},
		DeployedAt: time.Now().UTC().Truncate(time.Second),
	}

	body, err := jsonapi.Marshal(want)
	if err != nil {
		t.Fatalf("jsonapi.Marshal: %v", err)
	}

	var got RestModel
	if err := jsonapi.Unmarshal(body, &got); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if len(got.Equipment) != len(want.Equipment) {
		t.Fatalf("Equipment len = %d, want %d (body: %s)", len(got.Equipment), len(want.Equipment), body)
	}
	for i, e := range got.Equipment {
		if e.Slot != want.Equipment[i].Slot || e.ItemId != want.Equipment[i].ItemId {
			t.Errorf("Equipment[%d] = %+v, want %+v", i, e, want.Equipment[i])
		}
	}

	m, err := Extract(got)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	eq := m.Equipment()
	if len(eq) != 3 {
		t.Fatalf("Extract().Equipment() len = %d, want 3", len(eq))
	}
	for i, e := range eq {
		if e.Slot() != want.Equipment[i].Slot || e.ItemId() != want.Equipment[i].ItemId {
			t.Errorf("Extract().Equipment()[%d] = (%d, %d), want (%d, %d)", i, e.Slot(), e.ItemId(), want.Equipment[i].Slot, want.Equipment[i].ItemId)
		}
	}
}

// TestForEachInMap_RequestsByMapAndWorld asserts ForEachInMap fetches
// player-npcs?filter[mapId]=%d&filter[worldId]=%d (requests.go's Resource
// template) against the field's map and world.
func TestForEachInMap_RequestsByMapAndWorld(t *testing.T) {
	const wantMapId = _map.Id(100000000)
	const wantWorldId = world.Id(0)

	var gotMapId, gotWorldId string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotMapId = q.Get("filter[mapId]")
		gotWorldId = q.Get("filter[worldId]")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "player-npcs",
					"id": "0eaf71e9-2ba7-4443-96bb-886f7dc8213c",
					"attributes": {
						"characterId": 42,
						"name": "Bowman",
						"worldId": 0,
						"mapId": 100000000,
						"scriptId": 9000000,
						"objectId": 7,
						"gender": 1,
						"skin": 0,
						"face": 20000,
						"hair": 30000,
						"jobId": 100,
						"x": 250,
						"cy": 300,
						"fh": 12,
						"rx0": 200,
						"rx1": 300,
						"dir": 1,
						"worldRank": 1,
						"overallRank": 2,
						"worldJobRank": 3,
						"overallJobRank": 4,
						"equipment": [],
						"deployedAt": "2024-01-01T00:00:00Z"
					}
				}
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(wantWorldId, channel.Id(0), wantMapId).Build()

	var seen []Model
	err := NewProcessor(logrus.New(), ctx).ForEachInMap(f, func(m Model) error {
		seen = append(seen, m)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachInMap: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("seen = %d models, want 1", len(seen))
	}
	if gotMapId != strconv.Itoa(int(wantMapId)) {
		t.Errorf("filter[mapId] = %q, want %q", gotMapId, strconv.Itoa(int(wantMapId)))
	}
	if gotWorldId != strconv.Itoa(int(wantWorldId)) {
		t.Errorf("filter[worldId] = %q, want %q", gotWorldId, strconv.Itoa(int(wantWorldId)))
	}
}
