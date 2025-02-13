package statup

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Type:   m.buffType,
		Amount: m.amount,
	}, nil
}
