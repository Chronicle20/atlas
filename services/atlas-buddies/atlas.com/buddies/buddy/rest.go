package buddy

import (
	"strconv"
)

type RestModel struct {
	CharacterId   uint32 `json:"characterId"`
	Group         string `json:"group"`
	CharacterName string `json:"characterName"`
	ChannelId     byte   `json:"channelId"`
	Pending       bool   `json:"pending"`
}

func (r RestModel) GetName() string {
	return "buddies"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.CharacterId))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.CharacterId = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CharacterId:   m.characterId,
		Group:         m.group,
		CharacterName: m.characterName,
		ChannelId:     m.channelId,
		Pending:       m.pending,
	}, nil
}
