package area_info

// RestModel is the JSON:API resource for a character's area-info entry.
type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	Area        uint16 `json:"area"`
	Info        string `json:"info"`
}

func (r RestModel) GetName() string {
	return "area-infos"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Extract converts a RestModel to a domain Model.
func Extract(r RestModel) (Model, error) {
	return Model{
		characterId: r.CharacterId,
		area:        r.Area,
		info:        r.Info,
	}, nil
}
