package information

import "testing"

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
