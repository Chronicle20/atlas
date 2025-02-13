package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterAttackMagic = "CharacterAttackMagic"

func CharacterAttackMagicBody(l logrus.FieldLogger, t tenant.Model) func(c character.Model, ai model.AttackInfo) BodyProducer {
	return func(c character.Model, ai model.AttackInfo) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			WriteCommonAttackBody(t)(c, ai)(w)
			return w.Bytes()
		}
	}
}
