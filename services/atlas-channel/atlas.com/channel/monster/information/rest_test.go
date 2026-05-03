package information

import "testing"

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
