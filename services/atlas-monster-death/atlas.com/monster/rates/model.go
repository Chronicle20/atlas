package rates

// Model holds computed rates for a character
type Model struct {
	expRate      float64
	mesoRate     float64
	itemDropRate float64
	questExpRate float64
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

// Default returns rates with all multipliers at 1.0
func Default() Model {
	return Model{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}
