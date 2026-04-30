package preset

import (
	"encoding/json"
	"testing"
)

func TestRestModel_RoundTrip(t *testing.T) {
	in := RestModel{
		Id: "5e1c0b6e-8a52-4c33-9f4a-6c2c1bc9c1d7",
		Attributes: Attributes{
			Name:      "Hero — 4th job",
			JobId:     112,
			Gender:    0,
			Level:     200,
			Stats:     StatBlock{Str: 999, Hp: 30000, Mp: 6000},
			Equipment: []EquipmentEntry{{TemplateId: 1002357, UseAverageStats: true}},
			Inventory: []InventoryEntry{{TemplateId: 2000000, Quantity: 200}},
			Skills:    []SkillEntry{{SkillId: 1121008, Level: 30}},
		},
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out RestModel
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Attributes.Equipment[0].TemplateId != 1002357 {
		t.Fatalf("equipment templateId did not survive round trip")
	}
	if !out.Attributes.Equipment[0].UseAverageStats {
		t.Fatalf("UseAverageStats flag lost")
	}
	if out.Attributes.Stats.Hp != 30000 {
		t.Fatalf("stat hp lost")
	}
	if len(out.Attributes.Skills) != 1 || out.Attributes.Skills[0].Level != 30 {
		t.Fatalf("skill entry lost")
	}
}
