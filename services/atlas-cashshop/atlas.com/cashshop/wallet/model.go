package wallet

import "github.com/google/uuid"

type Model struct {
	id          uuid.UUID
	characterId uint32
	credit      uint32
	points      uint32
	prepaid     uint32
}
