package writer

import (
	"atlas-channel/character/buff"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterBuffGive = "CharacterBuffGive"
const CharacterBuffGiveForeign = "CharacterBuffGiveForeign"

func CharacterBuffGiveBody(l logrus.FieldLogger) func(ctx context.Context) func(buffs []buff.Model) BodyProducer {
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
				cts.Encode(l, t, options)(w)
				w.WriteShort(0) // tDelay
				w.WriteByte(0)  // MovementAffectingStat
				return w.Bytes()
			}
		}
	}
}

func CharacterBuffGiveForeignBody(l logrus.FieldLogger) func(ctx context.Context) func(fromId uint32, buffs []buff.Model) BodyProducer {
	return func(ctx context.Context) func(fromId uint32, buffs []buff.Model) BodyProducer {
		t := tenant.MustFromContext(ctx)
		return func(fromId uint32, buffs []buff.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				w.WriteInt(fromId)
				cts := model.NewCharacterTemporaryStat()
				for _, b := range buffs {
					for _, c := range b.Changes() {
						cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.ExpiresAt())
					}
				}
				cts.EncodeForeign(l, t, options)(w)
				w.WriteShort(0) // tDelay
				w.WriteByte(0)  // MovementAffectingStat
				return w.Bytes()
			}
		}
	}
}
