package information

import "testing"

func TestExtract_PopulatesAttacks(t *testing.T) {
	rm := RestModel{
		Id:      "5100004",
		Hp:      3000,
		Mp:      100,
		Attacks: []AttackInfoRestModel{{Pos: 2, ConMP: 5, AttackAfter: 1500}},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(m.Attacks()) != 1 {
		t.Fatalf("Attacks length = %d, want 1", len(m.Attacks()))
	}
	got := m.Attacks()[0]
	if got.Pos != 2 || got.ConMP != 5 || got.AttackAfter != 1500 {
		t.Fatalf("Attack[0] = %+v, want {Pos:2 ConMP:5 AttackAfter:1500}", got)
	}
}

func TestExtract_PopulatesRecoveryFields(t *testing.T) {
	rm := RestModel{
		Id:         "100100",
		Hp:         1000,
		Mp:         100,
		HpRecovery: 20,
		MpRecovery: 5,
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.HpRecovery() != 20 {
		t.Errorf("HpRecovery: got %d, want 20", m.HpRecovery())
	}
	if m.MpRecovery() != 5 {
		t.Errorf("MpRecovery: got %d, want 5", m.MpRecovery())
	}
}
