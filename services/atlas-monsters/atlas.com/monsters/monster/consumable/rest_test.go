package consumable

import (
	"reflect"
	"testing"
)

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
	m := NewBuilder().SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build()
	if m.MonsterHp() != 0 || m.BridleProp() != 0 {
		t.Fatalf("unset fields must be zero: %+v", m)
	}
	if m.Id() != 2270000 || m.MonsterId() != 9300101 || m.Create() != 1902000 {
		t.Fatalf("builder produced %+v", m)
	}
}

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives an Extract -> Transform ->
// Extract round trip, with distinct non-zero values for every field.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:            1,
		Create:        2,
		MonsterId:     3,
		MonsterHP:     4,
		BridleProp:    5,
		BridlePropChg: 6.5,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot      %+v", m, m2)
	}
}
