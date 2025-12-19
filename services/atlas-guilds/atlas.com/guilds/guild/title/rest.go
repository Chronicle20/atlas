package title

type RestModel struct {
	Name  string `json:"name"`
	Index byte   `json:"index"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Name:  m.name,
		Index: m.index,
	}, nil
}
