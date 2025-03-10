package writer

import (
	"atlas-channel/pet"
	model2 "atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetActivated = "PetActivated"

func PetSpawnBody(l logrus.FieldLogger) func(t tenant.Model) func(characterId uint32, p pet.Model) BodyProducer {
	return func(t tenant.Model) func(characterId uint32, p pet.Model) BodyProducer {
		return func(characterId uint32, p pet.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				w.WriteInt(characterId)
				w.WriteByte(p.Slot() - 1)
				w.WriteBool(true)
				w.WriteBool(true) // show?
				m := model2.Pet{
					TemplateId:  p.TemplateId(),
					Name:        p.Name(),
					Id:          p.Id(),
					X:           p.X(),
					Y:           p.Y(),
					Stance:      p.Stance(),
					Foothold:    34,
					NameTag:     0,
					ChatBalloon: 0,
				}
				m.Encode(l, t, options)(w)
				return w.Bytes()
			}
		}
	}
}

func PetDespawnBody(l logrus.FieldLogger) func(characterId uint32, p pet.Model) BodyProducer {
	return func(characterId uint32, p pet.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(p.Slot())
			w.WriteBool(false)
			return w.Bytes()
		}
	}
}
