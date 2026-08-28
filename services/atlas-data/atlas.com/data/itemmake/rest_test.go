package itemmake

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestRestModelGetName(t *testing.T) {
	if got := (RestModel{}).GetName(); got != "itemMakes" {
		t.Fatalf("expected GetName() to return %q, got %q", "itemMakes", got)
	}
}

func TestRestModelIdRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		id   uint32
	}{
		{"group zero crystal", 4260000},
		{"equip recipe", 1082002},
		{"zero", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := RestModel{Id: tc.id}
			expected := strconv.Itoa(int(tc.id))
			if got := in.GetID(); got != expected {
				t.Fatalf("expected GetID() to return %q, got %q", expected, got)
			}

			var out RestModel
			if err := out.SetID(expected); err != nil {
				t.Fatalf("unexpected error from SetID: %v", err)
			}
			if out.Id != tc.id {
				t.Fatalf("expected SetID to set Id to %d, got %d", tc.id, out.Id)
			}
		})
	}
}

func TestRestModelJSONPreservesListOrder(t *testing.T) {
	in := RestModel{
		Id:    1082002,
		Group: 1,
		Recipe: []MaterialRestModel{
			{ItemId: 4011001, Count: 5},
			{ItemId: 4011002, Count: 3},
			{ItemId: 4021007, Count: 1},
		},
		RandomReward: []RewardRestModel{
			{ItemId: 4260000, ItemNum: 1, Prob: 70},
			{ItemId: 4260001, ItemNum: 1, Prob: 25},
			{ItemId: 4260002, ItemNum: 1, Prob: 5},
		},
		ReqQuest: []QuestReqRestModel{{QuestId: 21614, State: 3}},
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("unexpected error marshaling: %v", err)
	}

	var out RestModel
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}

	if len(out.Recipe) != 3 {
		t.Fatalf("expected 3 recipe entries, got %d", len(out.Recipe))
	}
	if out.Recipe[0].ItemId != 4011001 {
		t.Fatalf("expected Recipe[0].ItemId == 4011001, got %d", out.Recipe[0].ItemId)
	}
	if out.Recipe[1].ItemId != 4011002 {
		t.Fatalf("expected Recipe[1].ItemId == 4011002, got %d", out.Recipe[1].ItemId)
	}
	if out.Recipe[2].ItemId != 4021007 {
		t.Fatalf("expected Recipe[2].ItemId == 4021007, got %d", out.Recipe[2].ItemId)
	}

	if len(out.RandomReward) != 3 {
		t.Fatalf("expected 3 randomReward entries, got %d", len(out.RandomReward))
	}
	if out.RandomReward[0].Prob != 70 {
		t.Fatalf("expected RandomReward[0].Prob == 70, got %d", out.RandomReward[0].Prob)
	}
	if out.RandomReward[1].Prob != 25 {
		t.Fatalf("expected RandomReward[1].Prob == 25, got %d", out.RandomReward[1].Prob)
	}
	if out.RandomReward[2].Prob != 5 {
		t.Fatalf("expected RandomReward[2].Prob == 5, got %d", out.RandomReward[2].Prob)
	}

	if len(out.ReqQuest) != 1 {
		t.Fatalf("expected 1 reqQuest entry, got %d", len(out.ReqQuest))
	}
	if out.ReqQuest[0].QuestId != 21614 || out.ReqQuest[0].State != 3 {
		t.Fatalf("expected ReqQuest[0] == {21614 3}, got %+v", out.ReqQuest[0])
	}
}

func TestRestModelAbsentListsAreEmptyNotNilOnRoundTrip(t *testing.T) {
	raw := `{"group":0,"reqLevel":0,"reqSkillLevel":0,"itemNum":0,"tuc":0,"meso":0,"catalyst":0,"reqItem":0,"reqEquip":0}`

	var out RestModel
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}

	if out.Group != 0 || out.ReqLevel != 0 || out.ReqSkillLevel != 0 || out.ItemNum != 0 ||
		out.Tuc != 0 || out.Meso != 0 || out.Catalyst != 0 || out.ReqItem != 0 || out.ReqEquip != 0 {
		t.Fatalf("expected all scalar fields to be 0, got %+v", out)
	}

	if len(out.Recipe) != 0 {
		t.Fatalf("expected Recipe to be empty, got %d entries", len(out.Recipe))
	}
	if len(out.RandomReward) != 0 {
		t.Fatalf("expected RandomReward to be empty, got %d entries", len(out.RandomReward))
	}
	if len(out.ReqQuest) != 0 {
		t.Fatalf("expected ReqQuest to be empty, got %d entries", len(out.ReqQuest))
	}
}

func TestGetModelRegistryIsSingleton(t *testing.T) {
	a := GetModelRegistry()
	b := GetModelRegistry()
	if a != b {
		t.Fatalf("expected GetModelRegistry() to return the same pointer on both calls, got %p and %p", a, b)
	}
}
