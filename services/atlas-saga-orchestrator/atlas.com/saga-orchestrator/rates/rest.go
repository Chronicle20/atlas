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

// Transform is the exact inverse of Extract: it converts a Model back into
// the REST DataBody payload.
func Transform(m Model) DataBody {
	return DataBody{
		Attributes: Attributes{
			ExpRate:      m.expRate,
			MesoRate:     m.mesoRate,
			ItemDropRate: m.itemDropRate,
			QuestExpRate: m.questExpRate,
		},
	}
}
