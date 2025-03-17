package statup

type Model struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func NewModel(buffType string, amount int32) Model {
	return Model{
		Type:   buffType,
		Amount: amount,
	}
}
