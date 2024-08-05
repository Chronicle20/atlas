package _map

import "github.com/google/uuid"

type MapKey struct {
	TenantId  uuid.UUID
	WorldId   byte
	ChannelId byte
	MapId     uint32
}
