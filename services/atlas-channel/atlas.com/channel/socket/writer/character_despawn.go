package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterDespawn = "CharacterDespawn"

func CharacterDespawnBody(l logrus.FieldLogger, t tenant.Model) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return w.Bytes()
		}
	}
}
