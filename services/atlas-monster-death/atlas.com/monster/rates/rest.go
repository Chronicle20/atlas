package rates

type RestModel struct {
	Id           string  `json:"-"`
	ExpRate      float64 `json:"expRate"`
	MesoRate     float64 `json:"mesoRate"`
	ItemDropRate float64 `json:"itemDropRate"`
	QuestExpRate float64 `json:"questExpRate"`
}

func (r RestModel) GetName() string {
	return "rates"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		expRate:      rm.ExpRate,
		mesoRate:     rm.MesoRate,
		itemDropRate: rm.ItemDropRate,
		questExpRate: rm.QuestExpRate,
	}, nil
}
