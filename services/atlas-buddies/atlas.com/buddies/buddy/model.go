package buddy

import "github.com/google/uuid"

type Model struct {
	listId        uuid.UUID
	characterId   uint32
	group         string
	characterName string
	channelId     int8
	inShop        bool
	pending       bool
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Name() string {
	return m.characterName
}

func (m Model) Group() string {
	return m.group
}

func (m Model) ChannelId() int8 {
	return m.channelId
}
