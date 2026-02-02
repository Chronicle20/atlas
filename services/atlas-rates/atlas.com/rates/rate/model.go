package rate

// Type identifies what the rate affects
type Type string

const (
	TypeExp      Type = "exp"
	TypeMeso     Type = "meso"
	TypeItemDrop Type = "item_drop"
	TypeQuestExp Type = "quest_exp"
)

// AllTypes returns all defined rate types
func AllTypes() []Type {
	return []Type{TypeExp, TypeMeso, TypeItemDrop, TypeQuestExp}
}

// Factor represents a single contribution to a rate
type Factor struct {
	source     string  // e.g., "world", "channel", "buff:2311003", "item:1234567"
	rateType   Type    // which rate this factor affects
	multiplier float64 // the multiplier value (1.0 = no change)
}

func (f Factor) Source() string {
	return f.source
}

func (f Factor) RateType() Type {
	return f.rateType
}

func (f Factor) Multiplier() float64 {
	return f.multiplier
}

// NewFactor creates a new rate factor
func NewFactor(source string, rateType Type, multiplier float64) Factor {
	return Factor{
		source:     source,
		rateType:   rateType,
		multiplier: multiplier,
	}
}

// Computed holds all computed rates for a character
type Computed struct {
	expRate      float64
	mesoRate     float64
	itemDropRate float64
	questExpRate float64
}

func (c Computed) ExpRate() float64 {
	return c.expRate
}

func (c Computed) MesoRate() float64 {
	return c.mesoRate
}

func (c Computed) ItemDropRate() float64 {
	return c.itemDropRate
}

func (c Computed) QuestExpRate() float64 {
	return c.questExpRate
}

// GetRate returns the rate for a specific type
func (c Computed) GetRate(t Type) float64 {
	switch t {
	case TypeExp:
		return c.expRate
	case TypeMeso:
		return c.mesoRate
	case TypeItemDrop:
		return c.itemDropRate
	case TypeQuestExp:
		return c.questExpRate
	default:
		return 1.0
	}
}

// NewComputed creates a new computed rates model
func NewComputed(expRate, mesoRate, itemDropRate, questExpRate float64) Computed {
	return Computed{
		expRate:      expRate,
		mesoRate:     mesoRate,
		itemDropRate: itemDropRate,
		questExpRate: questExpRate,
	}
}

// DefaultComputed returns computed rates with all values at 1.0
func DefaultComputed() Computed {
	return Computed{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

// ComputeFromFactors aggregates factors into computed rates
func ComputeFromFactors(factors []Factor) Computed {
	// Start with base rate of 1.0 for all types
	rates := map[Type]float64{
		TypeExp:      1.0,
		TypeMeso:     1.0,
		TypeItemDrop: 1.0,
		TypeQuestExp: 1.0,
	}

	// Multiply all factors for each type
	for _, f := range factors {
		if current, ok := rates[f.rateType]; ok {
			rates[f.rateType] = current * f.multiplier
		}
	}

	return Computed{
		expRate:      rates[TypeExp],
		mesoRate:     rates[TypeMeso],
		itemDropRate: rates[TypeItemDrop],
		questExpRate: rates[TypeQuestExp],
	}
}
