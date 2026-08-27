package maplelife

import (
	"encoding/json"
	"reflect"
	"testing"
)

func fullRestModel() RestModel {
	return RestModel{
		Looks: []LookOptions{
			{
				Gender:     0,
				Faces:      []uint32{20000, 20001},
				Hairs:      []uint32{30000, 30001},
				HairColors: []uint32{0, 1, 2},
				SkinColors: []uint32{0, 1},
			},
			{
				Gender:     1,
				Faces:      []uint32{21000, 21001},
				Hairs:      []uint32{31000, 31001},
				HairColors: []uint32{0, 1, 2},
				SkinColors: []uint32{0, 1},
			},
		},
		Classes: []ClassEntry{
			{
				Ordinal: 0,
				Gender:  0,
				JobId:   100,
				Level:   10,
				MapId:   10000,
				Stats: StatBlock{
					Str: 35,
					Dex: 20,
					Int: 4,
					Luk: 4,
					Hp:  100,
					Mp:  50,
				},
				AP:        5,
				SP:        "1,0,0,0,0,0,0,0,0,0",
				SpSkillId: 1000001,
				Meso:      1000,
				Equipment: []EquipmentEntry{
					{TemplateId: 1040002, UseAverageStats: true},
				},
				Inventory: []InventoryEntry{
					{TemplateId: 2000000, Quantity: 10},
				},
			},
			{
				Ordinal: 0,
				Gender:  1,
				JobId:   100,
				Level:   10,
				MapId:   10000,
				Stats: StatBlock{
					Str: 35,
					Dex: 20,
					Int: 4,
					Luk: 4,
					Hp:  100,
					Mp:  50,
				},
				AP:        5,
				SP:        "1,0,0,0,0,0,0,0,0,0",
				SpSkillId: 1000001,
				Meso:      1000,
				Equipment: []EquipmentEntry{
					{TemplateId: 1041002, UseAverageStats: true},
				},
				Inventory: []InventoryEntry{
					{TemplateId: 2000000, Quantity: 10},
				},
			},
		},
	}
}

func TestMapleLifeBlockRoundTrips(t *testing.T) {
	t.Run("full document", func(t *testing.T) {
		input := fullRestModel()

		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var decoded RestModel
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !reflect.DeepEqual(input, decoded) {
			t.Errorf("round trip mismatch:\ninput:   %+v\ndecoded: %+v", input, decoded)
		}
	})

	t.Run("absent block", func(t *testing.T) {
		var decoded RestModel
		if err := json.Unmarshal([]byte(`{}`), &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if len(decoded.Looks) != 0 {
			t.Errorf("expected 0 Looks, got %d", len(decoded.Looks))
		}
		if len(decoded.Classes) != 0 {
			t.Errorf("expected 0 Classes, got %d", len(decoded.Classes))
		}
	})

	t.Run("json tags", func(t *testing.T) {
		input := fullRestModel()

		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("unmarshal into map failed: %v", err)
		}

		if _, ok := raw["looks"]; !ok {
			t.Errorf("expected top-level key 'looks'")
		}
		classesRaw, ok := raw["classes"]
		if !ok {
			t.Fatalf("expected top-level key 'classes'")
		}

		classes, ok := classesRaw.([]interface{})
		if !ok || len(classes) == 0 {
			t.Fatalf("expected non-empty classes array")
		}

		class, ok := classes[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected class entry to be an object")
		}

		expectedKeys := []string{
			"ordinal", "gender", "jobId", "level", "mapId", "stats",
			"ap", "sp", "spSkillId", "meso", "equipment", "inventory",
		}
		for _, key := range expectedKeys {
			if _, ok := class[key]; !ok {
				t.Errorf("expected class entry to have key %q", key)
			}
		}
	})
}
