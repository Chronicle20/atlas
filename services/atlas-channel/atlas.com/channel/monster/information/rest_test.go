package information

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id: "5100004",
		Attacks: []AttackInfoRestModel{
			{Pos: 2, ConMP: 5, AttackAfter: 1500},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}

func TestExtract_PopulatesAttacks(t *testing.T) {
	rm := RestModel{
		Id: "5100004",
		Attacks: []AttackInfoRestModel{
			{Pos: 2, ConMP: 5, AttackAfter: 1500},
		},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(m.Attacks()) != 1 {
		t.Fatalf("Attacks = %d, want 1", len(m.Attacks()))
	}
	if m.Attacks()[0].Pos != 2 || m.Attacks()[0].ConMP != 5 || m.Attacks()[0].AttackAfter != 1500 {
		t.Fatalf("Attack[0] = %+v", m.Attacks()[0])
	}
}
