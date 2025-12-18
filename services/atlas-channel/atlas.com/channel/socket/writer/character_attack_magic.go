package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterAttackMagic = "CharacterAttackMagic"

func CharacterAttackMagicBody(l logrus.FieldLogger) func(ctx context.Context) func(c character.Model, ai model.AttackInfo) BodyProducer {
	return func(ctx context.Context) func(c character.Model, ai model.AttackInfo) BodyProducer {
		return func(c character.Model, ai model.AttackInfo) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				WriteCommonAttackBody(l)(ctx)(c, ai)(w)
				return w.Bytes()
			}
		}
	}
}
