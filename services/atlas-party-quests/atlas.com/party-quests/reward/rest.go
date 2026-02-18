package reward

type WeightedItemRestModel struct {
	TemplateId uint32 `json:"templateId"`
	Weight     uint32 `json:"weight"`
	Quantity   uint32 `json:"quantity"`
}

type RestModel struct {
	Type   string                  `json:"type"`
	Amount uint32                  `json:"amount,omitempty"`
	Items  []WeightedItemRestModel `json:"items,omitempty"`
}

func Transform(m Model) (RestModel, error) {
	items := make([]WeightedItemRestModel, 0, len(m.Items()))
	for _, item := range m.Items() {
		items = append(items, WeightedItemRestModel{
			TemplateId: item.TemplateId(),
			Weight:     item.Weight(),
			Quantity:   item.Quantity(),
		})
	}
	return RestModel{
		Type:   m.Type(),
		Amount: m.Amount(),
		Items:  items,
	}, nil
}

func Extract(r RestModel) (Model, error) {
	builder := NewBuilder().
		SetType(r.Type).
		SetAmount(r.Amount)

	for _, item := range r.Items {
		wi, err := NewWeightedItemBuilder().
			SetTemplateId(item.TemplateId).
			SetWeight(item.Weight).
			SetQuantity(item.Quantity).
			Build()
		if err != nil {
			return Model{}, err
		}
		builder.AddItem(wi)
	}

	return builder.Build()
}
