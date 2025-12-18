package stat

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		statType: rm.Type,
		amount:   rm.Amount,
	}, nil
}
