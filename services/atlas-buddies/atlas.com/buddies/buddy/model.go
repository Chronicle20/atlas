package buddy

import "github.com/google/uuid"

type Model struct {
	id            uuid.UUID
	listId        uuid.UUID
	characterId   uint32
	group         string
	characterName string
	channelId     byte
	visible       bool
}
