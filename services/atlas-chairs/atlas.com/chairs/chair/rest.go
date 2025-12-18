package chair

import "strconv"

type RestModel struct {
	Id          uint32 `json:"-"`
	Type        string `json:"type"`
	CharacterId uint32 `json:"characterId"`
}

func (r RestModel) GetName() string {
	return "chairs"
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

func Transform(characterId uint32) func(m Model) (RestModel, error) {
	return func(m Model) (RestModel, error) {
		return RestModel{
			Id:          m.id,
			Type:        m.chairType,
			CharacterId: characterId,
		}, nil
	}
}
