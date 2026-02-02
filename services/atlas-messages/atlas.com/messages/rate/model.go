package rate

// Factor represents a single contribution to a rate
type Factor struct {
	source     string
	rateType   string
	multiplier float64
}

func (f Factor) Source() string {
	return f.source
}

func (f Factor) RateType() string {
	return f.rateType
}

func (f Factor) Multiplier() float64 {
	return f.multiplier
}

// Model holds computed rates and their factors for a character
type Model struct {
	characterId  uint32
	expRate      float64
	mesoRate     float64
	itemDropRate float64
	questExpRate float64
	factors      []Factor
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) ExpRate() float64 {
	return m.expRate
}

func (m Model) MesoRate() float64 {
	return m.mesoRate
}

func (m Model) ItemDropRate() float64 {
	return m.itemDropRate
}

func (m Model) QuestExpRate() float64 {
	return m.questExpRate
}

func (m Model) Factors() []Factor {
	return m.factors
}

// FactorsByType returns all factors for a specific rate type
func (m Model) FactorsByType(rateType string) []Factor {
	var result []Factor
	for _, f := range m.factors {
		if f.rateType == rateType {
			result = append(result, f)
		}
	}
	return result
}
