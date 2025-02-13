package statup

type Model struct {
	buffType string
	amount   int32
}

func (s Model) Mask() string {
	return s.buffType
}

func (s Model) Amount() int32 {
	return s.amount
}

func NewModel(buffType string, amount int32) Model {
	return Model{
		buffType: buffType,
		amount:   amount,
	}
}
