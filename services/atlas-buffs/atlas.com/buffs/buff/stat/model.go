package stat

type Model struct {
	statType string
	amount   int32
}

func (m Model) Type() string {
	return m.statType
}

func (m Model) Amount() int32 {
	return m.amount
}

func NewStat(statType string, amount int32) Model {
	return Model{
		statType: statType,
		amount:   amount,
	}
}
