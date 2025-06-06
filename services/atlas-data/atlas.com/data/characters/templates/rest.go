package templates

import "strconv"

type RestModel struct {
	Id            uint32   `json:"-"`
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
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
