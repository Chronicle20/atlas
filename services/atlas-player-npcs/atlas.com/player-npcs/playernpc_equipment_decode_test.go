package main

import (
	"atlas-player-npcs/playernpc"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// TestPlayerNpcRestModel_EquipmentRoundTrips closes the gap the Task 18
// review left open: playernpc.RestModel.Equipment is a plain (non-relationship)
// nested struct-slice attribute, and until now nothing actually ran it
// through real api2go/jsonapi Marshal+Unmarshal with a non-empty slice — both
// this service's own resource_test.go and atlas-channel's rest_test.go build
// the model directly and never exercise the wire decode. If this test fails,
// the nested equipment attribute does not survive the real decode path and
// every deployed player NPC's equipment is silently lost.
func TestPlayerNpcRestModel_EquipmentRoundTrips(t *testing.T) {
	want := playernpc.RestModel{
		Id:             uuid.New(),
		CharacterId:    1001,
		Name:           "Statue",
		WorldId:        0,
		MapId:          100000000,
		ScriptId:       9001,
		ObjectId:       1,
		Gender:         0,
		Skin:           0,
		Face:           20000,
		Hair:           30030,
		JobId:          0,
		X:              100,
		Cy:             200,
		Fh:             1,
		Rx0:            50,
		Rx1:            150,
		Dir:            1,
		WorldRank:      0,
		OverallRank:    0,
		WorldJobRank:   0,
		OverallJobRank: 0,
		Equipment: []playernpc.EquipmentRestModel{
			{Slot: -1, ItemId: 1302000},
			{Slot: -101, ItemId: 1040002},
		},
		DeployedAt: time.Now().UTC().Truncate(time.Second),
	}

	payload, err := jsonapi.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var got playernpc.RestModel
	if err := jsonapi.Unmarshal(payload, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if len(got.Equipment) != 2 {
		t.Fatalf("len(Equipment) = %d, want 2 (payload: %s)", len(got.Equipment), payload)
	}
	bySlot := make(map[int16]uint32, len(got.Equipment))
	for _, e := range got.Equipment {
		bySlot[e.Slot] = e.ItemId
	}
	if bySlot[-1] != 1302000 {
		t.Errorf("bySlot[-1] = %d, want 1302000", bySlot[-1])
	}
	if bySlot[-101] != 1040002 {
		t.Errorf("bySlot[-101] = %d, want 1040002", bySlot[-101])
	}
}
