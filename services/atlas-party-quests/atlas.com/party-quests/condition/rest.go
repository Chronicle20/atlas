package condition

type RestModel struct {
	Type         string `json:"type"`
	Operator     string `json:"operator"`
	Value        uint32 `json:"value"`
	ReferenceId  uint32 `json:"referenceId"`
	ReferenceKey string `json:"referenceKey,omitempty"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Type:         m.Type(),
		Operator:     m.Operator(),
		Value:        m.Value(),
		ReferenceId:  m.ReferenceId(),
		ReferenceKey: m.ReferenceKey(),
	}, nil
}

func Extract(r RestModel) (Model, error) {
	return NewBuilder().
		SetType(r.Type).
		SetOperator(r.Operator).
		SetValue(r.Value).
		SetReferenceId(r.ReferenceId).
		SetReferenceKey(r.ReferenceKey).
		Build()
}
