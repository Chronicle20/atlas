package area_info

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
