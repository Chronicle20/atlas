package stat

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Type:   m.Type(),
		Amount: m.Amount(),
	}, nil
}
