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
	PetDespawnModeNormal  = "NORMAL"
	PetDespawnModeHungry  = "HUNGER"
	PetDespawnModeExpired = "EXPIRED"
	PetDespawnModeUnk1    = "UNKNOWN_1"
	PetDespawnModeUnk2    = "UNKNOWN_2"
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

func PetDespawnBody(l logrus.FieldLogger) func(characterId uint32, slot int8, reason string) BodyProducer {
	return func(characterId uint32, slot int8, reason string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteInt8(slot)
			w.WriteBool(false)
			w.WriteByte(getPetDespawnOperation(l)(options, reason))
			return w.Bytes()
		}
	}
}

func getPetDespawnOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
