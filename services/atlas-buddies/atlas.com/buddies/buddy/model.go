package buddy

import "github.com/google/uuid"

type Model struct {
	listId        uuid.UUID
	characterId   uint32
	group         string
	characterName string
	channelId     byte
	pending       bool
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}
