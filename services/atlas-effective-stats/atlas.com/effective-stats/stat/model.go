package stat

// Type identifies which stat the bonus affects
type Type string

const (
	TypeStrength      Type = "strength"
	TypeDexterity     Type = "dexterity"
	TypeLuck          Type = "luck"
	TypeIntelligence  Type = "intelligence"
	TypeMaxHP         Type = "max_hp"
	TypeMaxMP         Type = "max_mp"
	TypeWeaponAttack  Type = "weapon_attack"
	TypeWeaponDefense Type = "weapon_defense"
	TypeMagicAttack   Type = "magic_attack"
	TypeMagicDefense  Type = "magic_defense"
	TypeAccuracy      Type = "accuracy"
	TypeAvoidability  Type = "avoidability"
	TypeSpeed         Type = "speed"
	TypeJump          Type = "jump"
)

// AllTypes returns all defined stat types
func AllTypes() []Type {
	return []Type{
		TypeStrength,
		TypeDexterity,
		TypeLuck,
		TypeIntelligence,
		TypeMaxHP,
		TypeMaxMP,
		TypeWeaponAttack,
		TypeWeaponDefense,
		TypeMagicAttack,
		TypeMagicDefense,
		TypeAccuracy,
		TypeAvoidability,
		TypeSpeed,
		TypeJump,
	}
}

// Bonus represents a single contribution to a stat
type Bonus struct {
	source     string  // e.g., "equipment:12345", "passive:1000001", "buff:2311003"
	statType   Type    // which stat this bonus affects
	amount     int32   // flat bonus value (+20)
	multiplier float64 // percentage bonus (1.10 = +10%, or 0.10 for additive multipliers)
}

func (b Bonus) Source() string {
	return b.source
}

func (b Bonus) StatType() Type {
	return b.statType
}

func (b Bonus) Amount() int32 {
	return b.amount
}

func (b Bonus) Multiplier() float64 {
	return b.multiplier
}

// NewBonus creates a new stat bonus with a flat amount
func NewBonus(source string, statType Type, amount int32) Bonus {
	return Bonus{
		source:     source,
		statType:   statType,
		amount:     amount,
		multiplier: 0.0,
	}
}

// NewMultiplierBonus creates a new stat bonus with a percentage multiplier
func NewMultiplierBonus(source string, statType Type, multiplier float64) Bonus {
	return Bonus{
		source:     source,
		statType:   statType,
		amount:     0,
		multiplier: multiplier,
	}
}

// NewFullBonus creates a new stat bonus with both flat amount and multiplier
func NewFullBonus(source string, statType Type, amount int32, multiplier float64) Bonus {
	return Bonus{
		source:     source,
		statType:   statType,
		amount:     amount,
		multiplier: multiplier,
	}
}

// Computed holds all computed effective stats for a character
type Computed struct {
	strength      uint32
	dexterity     uint32
	luck          uint32
	intelligence  uint32
	maxHP         uint32
	maxMP         uint32
	weaponAttack  uint32
	weaponDefense uint32
	magicAttack   uint32
	magicDefense  uint32
	accuracy      uint32
	avoidability  uint32
	speed         uint32
	jump          uint32
}

func (c Computed) Strength() uint32 {
	return c.strength
}

func (c Computed) Dexterity() uint32 {
	return c.dexterity
}

func (c Computed) Luck() uint32 {
	return c.luck
}

func (c Computed) Intelligence() uint32 {
	return c.intelligence
}

func (c Computed) MaxHP() uint32 {
	return c.maxHP
}

func (c Computed) MaxMP() uint32 {
	return c.maxMP
}

func (c Computed) WeaponAttack() uint32 {
	return c.weaponAttack
}

func (c Computed) WeaponDefense() uint32 {
	return c.weaponDefense
}

func (c Computed) MagicAttack() uint32 {
	return c.magicAttack
}

func (c Computed) MagicDefense() uint32 {
	return c.magicDefense
}

func (c Computed) Accuracy() uint32 {
	return c.accuracy
}

func (c Computed) Avoidability() uint32 {
	return c.avoidability
}

func (c Computed) Speed() uint32 {
	return c.speed
}

func (c Computed) Jump() uint32 {
	return c.jump
}

// GetStat returns the computed value for a specific stat type
func (c Computed) GetStat(t Type) uint32 {
	switch t {
	case TypeStrength:
		return c.strength
	case TypeDexterity:
		return c.dexterity
	case TypeLuck:
		return c.luck
	case TypeIntelligence:
		return c.intelligence
	case TypeMaxHP:
		return c.maxHP
	case TypeMaxMP:
		return c.maxMP
	case TypeWeaponAttack:
		return c.weaponAttack
	case TypeWeaponDefense:
		return c.weaponDefense
	case TypeMagicAttack:
		return c.magicAttack
	case TypeMagicDefense:
		return c.magicDefense
	case TypeAccuracy:
		return c.accuracy
	case TypeAvoidability:
		return c.avoidability
	case TypeSpeed:
		return c.speed
	case TypeJump:
		return c.jump
	default:
		return 0
	}
}

// NewComputed creates a new computed stats model
func NewComputed(
	strength, dexterity, luck, intelligence uint32,
	maxHP, maxMP uint32,
	weaponAttack, weaponDefense, magicAttack, magicDefense uint32,
	accuracy, avoidability, speed, jump uint32,
) Computed {
	return Computed{
		strength:      strength,
		dexterity:     dexterity,
		luck:          luck,
		intelligence:  intelligence,
		maxHP:         maxHP,
		maxMP:         maxMP,
		weaponAttack:  weaponAttack,
		weaponDefense: weaponDefense,
		magicAttack:   magicAttack,
		magicDefense:  magicDefense,
		accuracy:      accuracy,
		avoidability:  avoidability,
		speed:         speed,
		jump:          jump,
	}
}

// Base holds the base stats from character service
type Base struct {
	strength     uint16
	dexterity    uint16
	luck         uint16
	intelligence uint16
	maxHP        uint16
	maxMP        uint16
}

func (b Base) Strength() uint16 {
	return b.strength
}

func (b Base) Dexterity() uint16 {
	return b.dexterity
}

func (b Base) Luck() uint16 {
	return b.luck
}

func (b Base) Intelligence() uint16 {
	return b.intelligence
}

func (b Base) MaxHP() uint16 {
	return b.maxHP
}

func (b Base) MaxMP() uint16 {
	return b.maxMP
}

// NewBase creates a new base stats model
func NewBase(strength, dexterity, luck, intelligence, maxHP, maxMP uint16) Base {
	return Base{
		strength:     strength,
		dexterity:    dexterity,
		luck:         luck,
		intelligence: intelligence,
		maxHP:        maxHP,
		maxMP:        maxMP,
	}
}

// MapBuffStatType maps buff stat type strings to Type and indicates if it's a multiplier.
// Returns empty string for unknown buff types.
func MapBuffStatType(buffType string) (Type, bool) {
	switch buffType {
	case "WEAPON_ATTACK", "PAD":
		return TypeWeaponAttack, false
	case "MAGIC_ATTACK", "MAD":
		return TypeMagicAttack, false
	case "WEAPON_DEFENSE", "PDD":
		return TypeWeaponDefense, false
	case "MAGIC_DEFENSE", "MDD":
		return TypeMagicDefense, false
	case "ACCURACY", "ACC":
		return TypeAccuracy, false
	case "AVOIDABILITY", "AVOID", "EVA":
		return TypeAvoidability, false
	case "SPEED":
		return TypeSpeed, false
	case "JUMP":
		return TypeJump, false
	case "HYPER_BODY_HP":
		return TypeMaxHP, true
	case "HYPER_BODY_MP":
		return TypeMaxMP, true
	case "MAPLE_WARRIOR":
		// Maple Warrior affects all primary stats - we need to handle this specially
		// For now, return strength as the representative stat
		// The actual implementation should add bonuses for all 4 primary stats
		return TypeStrength, true
	default:
		return "", false
	}
}

// MapStatupType maps statup/passive skill stat type strings to Type.
// Returns empty string for unknown stat types.
func MapStatupType(statupType string) Type {
	switch statupType {
	case "PAD", "WEAPON_ATTACK":
		return TypeWeaponAttack
	case "MAD", "MAGIC_ATTACK":
		return TypeMagicAttack
	case "PDD", "WEAPON_DEFENSE":
		return TypeWeaponDefense
	case "MDD", "MAGIC_DEFENSE":
		return TypeMagicDefense
	case "ACC", "ACCURACY":
		return TypeAccuracy
	case "EVA", "AVOID", "AVOIDABILITY":
		return TypeAvoidability
	case "SPEED":
		return TypeSpeed
	case "JUMP":
		return TypeJump
	case "HP", "MAX_HP", "MHP":
		return TypeMaxHP
	case "MP", "MAX_MP", "MMP":
		return TypeMaxMP
	case "STR", "STRENGTH":
		return TypeStrength
	case "DEX", "DEXTERITY":
		return TypeDexterity
	case "INT", "INTELLIGENCE":
		return TypeIntelligence
	case "LUK", "LUCK":
		return TypeLuck
	default:
		return ""
	}
}
