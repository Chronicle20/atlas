package state

import "strconv"

type RestModel struct {
	Id          uint32 `json:"-"`
	CharacterId uint32 `json:"characterId"`
	QuestId     uint32 `json:"questId"`
	State       State  `json:"state"`
}

func (r RestModel) GetName() string {
	return "quest-status"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId: rm.CharacterId,
		questId:     rm.QuestId,
		state:       rm.State,
	}, nil
}
