package stat

import (
	"testing"
)

func TestNewBonus(t *testing.T) {
	b := NewBonus("equipment:12345", TypeStrength, 20)

	if b.Source() != "equipment:12345" {
		t.Errorf("Source() = %v, want equipment:12345", b.Source())
	}
	if b.StatType() != TypeStrength {
		t.Errorf("StatType() = %v, want %v", b.StatType(), TypeStrength)
	}
	if b.Amount() != 20 {
		t.Errorf("Amount() = %v, want 20", b.Amount())
	}
	if b.Multiplier() != 0.0 {
		t.Errorf("Multiplier() = %v, want 0.0", b.Multiplier())
	}
}

func TestNewMultiplierBonus(t *testing.T) {
	b := NewMultiplierBonus("buff:2311003", TypeStrength, 0.10)

	if b.Source() != "buff:2311003" {
		t.Errorf("Source() = %v, want buff:2311003", b.Source())
	}
	if b.StatType() != TypeStrength {
		t.Errorf("StatType() = %v, want %v", b.StatType(), TypeStrength)
	}
	if b.Amount() != 0 {
		t.Errorf("Amount() = %v, want 0", b.Amount())
	}
	if b.Multiplier() != 0.10 {
		t.Errorf("Multiplier() = %v, want 0.10", b.Multiplier())
	}
}

func TestNewFullBonus(t *testing.T) {
	b := NewFullBonus("passive:1000001", TypeMaxHP, 100, 0.05)

	if b.Source() != "passive:1000001" {
		t.Errorf("Source() = %v, want passive:1000001", b.Source())
	}
	if b.StatType() != TypeMaxHP {
		t.Errorf("StatType() = %v, want %v", b.StatType(), TypeMaxHP)
	}
	if b.Amount() != 100 {
		t.Errorf("Amount() = %v, want 100", b.Amount())
	}
	if b.Multiplier() != 0.05 {
		t.Errorf("Multiplier() = %v, want 0.05", b.Multiplier())
	}
}

func TestNewBase(t *testing.T) {
	base := NewBase(50, 40, 30, 25, 5000, 3000)

	if base.Strength() != 50 {
		t.Errorf("Strength() = %v, want 50", base.Strength())
	}
	if base.Dexterity() != 40 {
		t.Errorf("Dexterity() = %v, want 40", base.Dexterity())
	}
	if base.Luck() != 30 {
		t.Errorf("Luck() = %v, want 30", base.Luck())
	}
	if base.Intelligence() != 25 {
		t.Errorf("Intelligence() = %v, want 25", base.Intelligence())
	}
	if base.MaxHP() != 5000 {
		t.Errorf("MaxHP() = %v, want 5000", base.MaxHP())
	}
	if base.MaxMP() != 3000 {
		t.Errorf("MaxMP() = %v, want 3000", base.MaxMP())
	}
}

func TestNewComputed(t *testing.T) {
	computed := NewComputed(100, 80, 60, 50, 10000, 5000, 150, 200, 100, 150, 50, 30, 100, 100)

	if computed.Strength() != 100 {
		t.Errorf("Strength() = %v, want 100", computed.Strength())
	}
	if computed.Dexterity() != 80 {
		t.Errorf("Dexterity() = %v, want 80", computed.Dexterity())
	}
	if computed.Luck() != 60 {
		t.Errorf("Luck() = %v, want 60", computed.Luck())
	}
	if computed.Intelligence() != 50 {
		t.Errorf("Intelligence() = %v, want 50", computed.Intelligence())
	}
	if computed.MaxHP() != 10000 {
		t.Errorf("MaxHP() = %v, want 10000", computed.MaxHP())
	}
	if computed.MaxMP() != 5000 {
		t.Errorf("MaxMP() = %v, want 5000", computed.MaxMP())
	}
	if computed.WeaponAttack() != 150 {
		t.Errorf("WeaponAttack() = %v, want 150", computed.WeaponAttack())
	}
	if computed.WeaponDefense() != 200 {
		t.Errorf("WeaponDefense() = %v, want 200", computed.WeaponDefense())
	}
	if computed.MagicAttack() != 100 {
		t.Errorf("MagicAttack() = %v, want 100", computed.MagicAttack())
	}
	if computed.MagicDefense() != 150 {
		t.Errorf("MagicDefense() = %v, want 150", computed.MagicDefense())
	}
	if computed.Accuracy() != 50 {
		t.Errorf("Accuracy() = %v, want 50", computed.Accuracy())
	}
	if computed.Avoidability() != 30 {
		t.Errorf("Avoidability() = %v, want 30", computed.Avoidability())
	}
	if computed.Speed() != 100 {
		t.Errorf("Speed() = %v, want 100", computed.Speed())
	}
	if computed.Jump() != 100 {
		t.Errorf("Jump() = %v, want 100", computed.Jump())
	}
}

func TestComputedGetStat(t *testing.T) {
	computed := NewComputed(100, 80, 60, 50, 10000, 5000, 150, 200, 100, 150, 50, 30, 110, 120)

	tests := []struct {
		statType Type
		expected uint32
	}{
		{TypeStrength, 100},
		{TypeDexterity, 80},
		{TypeLuck, 60},
		{TypeIntelligence, 50},
		{TypeMaxHP, 10000},
		{TypeMaxMP, 5000},
		{TypeWeaponAttack, 150},
		{TypeWeaponDefense, 200},
		{TypeMagicAttack, 100},
		{TypeMagicDefense, 150},
		{TypeAccuracy, 50},
		{TypeAvoidability, 30},
		{TypeSpeed, 110},
		{TypeJump, 120},
	}

	for _, tt := range tests {
		t.Run(string(tt.statType), func(t *testing.T) {
			if got := computed.GetStat(tt.statType); got != tt.expected {
				t.Errorf("GetStat(%v) = %v, want %v", tt.statType, got, tt.expected)
			}
		})
	}
}

func TestComputedGetStat_InvalidType(t *testing.T) {
	computed := NewComputed(100, 80, 60, 50, 10000, 5000, 150, 200, 100, 150, 50, 30, 110, 120)

	if got := computed.GetStat("invalid"); got != 0 {
		t.Errorf("GetStat(invalid) = %v, want 0", got)
	}
}

func TestAllTypes(t *testing.T) {
	types := AllTypes()

	if len(types) != 14 {
		t.Errorf("AllTypes() length = %v, want 14", len(types))
	}

	expected := []Type{
		TypeStrength, TypeDexterity, TypeLuck, TypeIntelligence,
		TypeMaxHP, TypeMaxMP,
		TypeWeaponAttack, TypeWeaponDefense, TypeMagicAttack, TypeMagicDefense,
		TypeAccuracy, TypeAvoidability, TypeSpeed, TypeJump,
	}

	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("AllTypes()[%d] = %v, want %v", i, types[i], tt)
		}
	}
}

func TestMapBuffStatType(t *testing.T) {
	tests := []struct {
		input        string
		expectedType Type
		isMultiplier bool
	}{
		// Flat bonuses
		{"WEAPON_ATTACK", TypeWeaponAttack, false},
		{"PAD", TypeWeaponAttack, false},
		{"MAGIC_ATTACK", TypeMagicAttack, false},
		{"MAD", TypeMagicAttack, false},
		{"WEAPON_DEFENSE", TypeWeaponDefense, false},
		{"PDD", TypeWeaponDefense, false},
		{"MAGIC_DEFENSE", TypeMagicDefense, false},
		{"MDD", TypeMagicDefense, false},
		{"ACCURACY", TypeAccuracy, false},
		{"ACC", TypeAccuracy, false},
		{"AVOIDABILITY", TypeAvoidability, false},
		{"AVOID", TypeAvoidability, false},
		{"EVA", TypeAvoidability, false},
		{"SPEED", TypeSpeed, false},
		{"JUMP", TypeJump, false},
		// Multiplier bonuses
		{"HYPER_BODY_HP", TypeMaxHP, true},
		{"HYPER_BODY_MP", TypeMaxMP, true},
		{"MAPLE_WARRIOR", TypeStrength, true},
		// Unknown type
		{"UNKNOWN", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			statType, isMultiplier := MapBuffStatType(tt.input)
			if statType != tt.expectedType {
				t.Errorf("MapBuffStatType(%q) type = %v, want %v", tt.input, statType, tt.expectedType)
			}
			if isMultiplier != tt.isMultiplier {
				t.Errorf("MapBuffStatType(%q) isMultiplier = %v, want %v", tt.input, isMultiplier, tt.isMultiplier)
			}
		})
	}
}

func TestMapStatupType(t *testing.T) {
	tests := []struct {
		input    string
		expected Type
	}{
		// Weapon/Magic attack/defense
		{"PAD", TypeWeaponAttack},
		{"WEAPON_ATTACK", TypeWeaponAttack},
		{"MAD", TypeMagicAttack},
		{"MAGIC_ATTACK", TypeMagicAttack},
		{"PDD", TypeWeaponDefense},
		{"WEAPON_DEFENSE", TypeWeaponDefense},
		{"MDD", TypeMagicDefense},
		{"MAGIC_DEFENSE", TypeMagicDefense},
		// Secondary stats
		{"ACC", TypeAccuracy},
		{"ACCURACY", TypeAccuracy},
		{"EVA", TypeAvoidability},
		{"AVOID", TypeAvoidability},
		{"AVOIDABILITY", TypeAvoidability},
		{"SPEED", TypeSpeed},
		{"JUMP", TypeJump},
		// HP/MP
		{"HP", TypeMaxHP},
		{"MAX_HP", TypeMaxHP},
		{"MHP", TypeMaxHP},
		{"MP", TypeMaxMP},
		{"MAX_MP", TypeMaxMP},
		{"MMP", TypeMaxMP},
		// Primary stats
		{"STR", TypeStrength},
		{"STRENGTH", TypeStrength},
		{"DEX", TypeDexterity},
		{"DEXTERITY", TypeDexterity},
		{"INT", TypeIntelligence},
		{"INTELLIGENCE", TypeIntelligence},
		{"LUK", TypeLuck},
		{"LUCK", TypeLuck},
		// Unknown type
		{"UNKNOWN", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := MapStatupType(tt.input); got != tt.expected {
				t.Errorf("MapStatupType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
