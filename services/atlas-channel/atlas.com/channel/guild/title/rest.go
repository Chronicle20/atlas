package title

type RestModel struct {
	Name  string `json:"name"`
	Index byte   `json:"index"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		name:  rm.Name,
		index: rm.Index,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Name:  m.name,
		Index: m.index,
	}, nil
}
