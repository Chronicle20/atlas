package inventory

import "github.com/google/uuid"

type Model struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType byte
	cashId        uint64
}
