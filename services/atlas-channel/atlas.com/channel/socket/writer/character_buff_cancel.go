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

const CharacterBuffCancel = "CharacterBuffCancel"
const CharacterBuffCancelForeign = "CharacterBuffCancelForeign"

func CharacterBuffCancelBody(buffs []buff.Model) packet.Encode {
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
			cts.EncodeMask(l, t, options)(w)
			w.WriteByte(0) // tSwallowBuffTime
			return w.Bytes()
		}
	}
}

func CharacterBuffCancelForeignBody(characterId uint32, buffs []buff.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			cts := model.NewCharacterTemporaryStat()
			for _, b := range buffs {
				for _, c := range b.Changes() {
					cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			cts.EncodeMask(l, t, options)(w)
			w.WriteByte(0) // tSwallowBuffTime
			return w.Bytes()
		}
	}
}
