package writer

import (
	"atlas-channel/character/buff"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterBuffGive = "CharacterBuffGive"
const CharacterBuffGiveForeign = "CharacterBuffGiveForeign"

func CharacterBuffGiveBody(buffs []buff.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			cts := model.NewCharacterTemporaryStat()
			for _, b := range buffs {
				for _, c := range b.Changes() {
					cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			w.WriteByteArray(cts.Encoder(l, ctx)(options))
			w.WriteShort(0) // tDelay
			w.WriteByte(0)  // MovementAffectingStat
			return w.Bytes()
		}
	}
}

func CharacterBuffGiveForeignBody(fromId uint32, buffs []buff.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(fromId)
			cts := model.NewCharacterTemporaryStat()
			for _, b := range buffs {
				for _, c := range b.Changes() {
					cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			w.WriteByteArray(cts.EncodeForeign(l, ctx)(options))
			w.WriteShort(0) // tDelay
			w.WriteByte(0)  // MovementAffectingStat
			return w.Bytes()
		}
	}
}
