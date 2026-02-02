package rate

type RestModel struct {
	Id           string  `json:"-"`
	RateType     string  `json:"rateType"`
	Multiplier   float64 `json:"multiplier"`
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

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		ExpRate:      m.ExpRate(),
		MesoRate:     m.MesoRate(),
		ItemDropRate: m.ItemDropRate(),
		QuestExpRate: m.QuestExpRate(),
	}, nil
}
