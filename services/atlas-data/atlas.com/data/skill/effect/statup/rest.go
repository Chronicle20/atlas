package statup

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Type:   m.Type,
		Amount: m.Amount,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		Type:   rm.Type,
		Amount: rm.Amount,
	}, nil
}
