package templates

type RestModel struct {
	Id            string   `json:"-"`
	CharacterType string   `json:"characterType"`
	Faces         []uint32 `json:"faces"`
	HairStyles    []uint32 `json:"hairStyles"`
	HairColors    []uint32 `json:"hairColors"`
	SkinColors    []uint32 `json:"skinColors"`
	Tops          []uint32 `json:"tops"`
	Bottoms       []uint32 `json:"bottoms"`
	Shoes         []uint32 `json:"shoes"`
	Weapons       []uint32 `json:"weapons"`
}

func (r RestModel) GetName() string {
	return "characterTemplates"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}
