package clientbound

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	PetDespawnModeNormal  = "NORMAL"
	PetDespawnModeHungry  = "HUNGER"
	PetDespawnModeExpired = "EXPIRED"
	PetDespawnModeUnk1    = "UNKNOWN_1"
	PetDespawnModeUnk2    = "UNKNOWN_2"
)

func PetSpawnBody(ownerId uint32, slot int8, templateId uint32, name string, petId uint64, x int16, y int16, stance byte, foothold uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return NewPetSpawnActivated(ownerId, slot, templateId, name, petId, x, y, stance, foothold).Encode
}

func PetDespawnBody(characterId uint32, slot int8, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", reason, func(mode byte) packet.Encoder {
		return NewPetDespawnActivated(characterId, slot, mode)
	})
}
