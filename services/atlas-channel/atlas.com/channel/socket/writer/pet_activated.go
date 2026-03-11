package writer

import (
	"atlas-channel/pet"
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


type PetDespawnMode byte

const (
	PetDespawnModeNormal  = "NORMAL"
	PetDespawnModeHungry  = "HUNGER"
	PetDespawnModeExpired = "EXPIRED"
	PetDespawnModeUnk1    = "UNKNOWN_1"
	PetDespawnModeUnk2    = "UNKNOWN_2"
)

func PetSpawnBody(p pet.Model) packet.Encode {
	return petpkt.NewPetSpawnActivated(p.OwnerId(), p.Slot(), p.TemplateId(), p.Name(), uint64(p.Id()), p.X(), p.Y(), p.Stance(), uint16(p.Fh())).Encode
}

func PetDespawnBody(characterId uint32, slot int8, reason string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPetDespawnOperation(l)(options, reason)
			return petpkt.NewPetDespawnActivated(characterId, slot, mode).Encode(l, ctx)(options)
		}
	}
}

func getPetDespawnOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
