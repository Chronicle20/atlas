package writer

import (
	"atlas-channel/pet"
	model2 "atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetActivated = "PetActivated"

type PetDespawnMode byte

const (
	PetDespawnModeNormal  = PetDespawnMode(0)
	PetDespawnModeHungry  = PetDespawnMode(1)
	PetDespawnModeExpired = PetDespawnMode(2)
	PetDespawnModeUnk1    = PetDespawnMode(3)
	PetDespawnModeUnk2    = PetDespawnMode(4)
)

func PetSpawnBody(l logrus.FieldLogger) func(t tenant.Model) func(p pet.Model) BodyProducer {
	return func(t tenant.Model) func(p pet.Model) BodyProducer {
		return func(p pet.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				w.WriteInt(p.OwnerId())
				w.WriteInt8(p.Slot())
				w.WriteBool(true)
				w.WriteBool(true) // show?
				m := model2.Pet{
					TemplateId:  p.TemplateId(),
					Name:        p.Name(),
					Id:          p.Id(),
					X:           p.X(),
					Y:           p.Y(),
					Stance:      p.Stance(),
					Foothold:    p.Fh(),
					NameTag:     0,
					ChatBalloon: 0,
				}
				m.Encode(l, t, options)(w)
				return w.Bytes()
			}
		}
	}
}

func PetDespawnBody(characterId uint32, slot int8, mode PetDespawnMode) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteInt8(slot)
		w.WriteBool(false)
		w.WriteByte(byte(mode))
		return w.Bytes()
	}
}
