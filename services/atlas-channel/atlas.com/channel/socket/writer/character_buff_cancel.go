package writer

import (
	"atlas-channel/character/buff"
	"context"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func CharacterBuffCancelBody(buffs []buff.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			cts := packetmodel.NewCharacterTemporaryStat()
			for _, b := range buffs {
				for _, c := range b.Changes() {
					cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			return charpkt.NewBuffCancel(*cts).Encode(l, ctx)(options)
		}
	}
}

func CharacterBuffCancelForeignBody(characterId uint32, buffs []buff.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			cts := packetmodel.NewCharacterTemporaryStat()
			for _, b := range buffs {
				for _, c := range b.Changes() {
					cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			return charpkt.NewBuffCancelForeign(characterId, *cts).Encode(l, ctx)(options)
		}
	}
}
