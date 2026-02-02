package rates

type DataListContainer struct {
	Data []DataBody `json:"data"`
}

type DataContainer struct {
	Data DataBody `json:"data"`
}

type DataBody struct {
	Id         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	ExpRate      float64 `json:"expRate"`
	MesoRate     float64 `json:"mesoRate"`
	ItemDropRate float64 `json:"itemDropRate"`
	QuestExpRate float64 `json:"questExpRate"`
}

func Extract(body DataBody) Model {
	return NewModel(
		body.Attributes.ExpRate,
		body.Attributes.MesoRate,
		body.Attributes.ItemDropRate,
		body.Attributes.QuestExpRate,
	)
}
