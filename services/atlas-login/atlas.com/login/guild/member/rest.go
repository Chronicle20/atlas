package member

type RestModel struct {
	CharacterId uint32 `json:"characterId"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId: rm.CharacterId,
	}, nil
}
