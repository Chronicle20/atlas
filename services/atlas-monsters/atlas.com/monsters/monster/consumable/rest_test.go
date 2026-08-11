package consumable

import "testing"

// TestExtract maps the atlas-data consumable resource onto the five fields the
// catch ladder reads. The upstream resource is much wider; this client is
// deliberately narrow.
func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:            2270002,
		Create:        4031868,
		MonsterId:     9300157,
		MonsterHP:     40,
		BridleProp:    50,
		BridlePropChg: 1.2,
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Id() != 2270002 || m.Create() != 4031868 || m.MonsterId() != 9300157 ||
		m.MonsterHp() != 40 || m.BridleProp() != 50 || m.BridlePropChg() != 1.2 {
		t.Fatalf("Extract produced %+v", m)
	}
}

// TestBuilder is the Builder-pattern seam the catch tests use for setup.
func TestBuilder(t *testing.T) {
	m := NewModelBuilder().SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build()
	if m.MonsterHp() != 0 || m.BridleProp() != 0 {
		t.Fatalf("unset fields must be zero: %+v", m)
	}
	if m.Id() != 2270000 || m.MonsterId() != 9300101 || m.Create() != 1902000 {
		t.Fatalf("builder produced %+v", m)
	}
}
