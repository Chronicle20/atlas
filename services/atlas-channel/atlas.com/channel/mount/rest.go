package mount

// RestModel mirrors atlas-mounts' JSON:API mount resource (type "mounts").
type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	Level       int    `json:"level"`
	Exp         int    `json:"exp"`
	Tiredness   int    `json:"tiredness"`
}

func (r RestModel) GetName() string { return "mounts" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId: rm.CharacterId,
		level:       rm.Level,
		exp:         rm.Exp,
		tiredness:   rm.Tiredness,
	}, nil
}
