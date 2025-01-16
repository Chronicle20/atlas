package buddy

import (
	"github.com/google/uuid"
)

type RestModel struct {
	Id            uuid.UUID `json:"id"`
	CharacterId   uint32    `json:"characterId"`
	Group         string    `json:"group"`
	CharacterName string    `json:"characterName"`
	ChannelId     byte      `json:"channelId"`
	Visible       bool      `json:"visible"`
}

func (r RestModel) GetName() string {
	return "buddies"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.id,
		CharacterId:   m.characterId,
		Group:         m.group,
		CharacterName: m.characterName,
		ChannelId:     m.channelId,
		Visible:       m.visible,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:            rm.Id,
		characterId:   rm.CharacterId,
		group:         rm.Group,
		characterName: rm.CharacterName,
		channelId:     rm.ChannelId,
		visible:       rm.Visible,
	}, nil
}
