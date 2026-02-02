package rate

type Type string

const (
	TypeExp      Type = "exp"
	TypeMeso     Type = "meso"
	TypeItemDrop Type = "item_drop"
	TypeQuestExp Type = "quest_exp"
)

type Model struct {
	expRate      float64
	mesoRate     float64
	itemDropRate float64
	questExpRate float64
}

func NewModel() Model {
	return Model{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
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

func (m Model) WithRate(rateType Type, multiplier float64) Model {
	switch rateType {
	case TypeExp:
		m.expRate = multiplier
	case TypeMeso:
		m.mesoRate = multiplier
	case TypeItemDrop:
		m.itemDropRate = multiplier
	case TypeQuestExp:
		m.questExpRate = multiplier
	}
	return m
}
