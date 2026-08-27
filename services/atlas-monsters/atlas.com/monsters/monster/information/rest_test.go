package information

import (
	"reflect"
	"testing"
)

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

// TestTransformRoundTrip asserts Extract(Transform(m)) reproduces every
// field Extract populates. Model is built as a literal (not via Builder)
// because Builder is intentionally a minimal test-fixture subset (per
// builder.go's doc comment) that omits hp, mp, undead, friendly,
// weaponAttack, dropPeriod, animationTimes, and revives.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		hp:              1000,
		mp:              500,
		boss:            true,
		undead:          true,
		friendly:        true,
		firstAttack:     true,
		weaponAttack:    200,
		dropPeriod:      30,
		resistances:     map[string]string{"P": "1", "I": "3"},
		animationTimes:  map[string]uint32{"die1": 1000, "hit1": 500},
		skills:          []Skill{{Id: 120, Level: 5}, {Id: 121, Level: 3}},
		revives:         []uint32{5100005, 5100006},
		banish:          Banish{Message: "begone", MapId: 100000000, PortalName: "sp"},
		attacks:         []AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}},
		selfDestruction: NewSelfDestruction(true, 1, 2000, 1500),
		hpRecovery:      20,
		mpRecovery:      5,
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}

func TestBuilderSetFirstAttack(t *testing.T) {
	if got := NewBuilder().SetFirstAttack(true).Build().FirstAttack(); got != true {
		t.Fatalf("SetFirstAttack(true).Build().FirstAttack() = %v, want true", got)
	}
	if got := NewBuilder().Build().FirstAttack(); got != false {
		t.Fatalf("Build().FirstAttack() zero value = %v, want false", got)
	}
}
