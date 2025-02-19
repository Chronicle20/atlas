package writer

import (
	"atlas-channel/character/buff"
	"atlas-channel/socket/model"
	"context"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterBuffCancel = "CharacterBuffCancel"

func CharacterBuffCancelBody(l logrus.FieldLogger) func(ctx context.Context) func(buffs []buff.Model) BodyProducer {
	return func(ctx context.Context) func(buffs []buff.Model) BodyProducer {
		t := tenant.MustFromContext(ctx)
		return func(buffs []buff.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				cts := model.NewCharacterTemporaryStat()
				for _, b := range buffs {
					for _, c := range b.Changes() {
						cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.ExpiresAt())
					}
				}
				cts.EncodeMask(l, t, options)(w)
				w.WriteByte(0) // tSwallowBuffTime
				return w.Bytes()
			}
		}
	}
}
