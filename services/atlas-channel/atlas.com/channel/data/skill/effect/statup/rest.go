package statup

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		buffType: rm.Type,
		amount:   rm.Amount,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Type:   m.buffType,
		Amount: m.amount,
	}, nil
}
