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

func TestExtractFirstAttack(t *testing.T) {
	tests := []struct {
		name string
		in   bool
		want bool
	}{
		{name: "aggressive template", in: true, want: true},
		{name: "passive template", in: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{FirstAttack: tt.in}
			got, err := Extract(rm)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			if got.FirstAttack() != tt.want {
				t.Fatalf("FirstAttack() = %v, want %v", got.FirstAttack(), tt.want)
			}
		})
	}
}

func TestModelBuilderSetFirstAttack(t *testing.T) {
	if got := NewModelBuilder().SetFirstAttack(true).Build().FirstAttack(); got != true {
		t.Fatalf("SetFirstAttack(true).Build().FirstAttack() = %v, want true", got)
	}
	if got := NewModelBuilder().Build().FirstAttack(); got != false {
		t.Fatalf("Build().FirstAttack() zero value = %v, want false", got)
	}
}
