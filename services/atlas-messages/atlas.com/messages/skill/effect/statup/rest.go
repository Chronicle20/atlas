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
