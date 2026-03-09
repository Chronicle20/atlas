package writer

import (
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterStatSet = "MonsterStatSet"
const MonsterStatReset = "MonsterStatReset"

func MonsterStatSetBody(uniqueId uint32, stat *model.MonsterTemporaryStat) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteByteArray(stat.Encode(l, ctx)(options))
			w.WriteInt16(0) // tDelay
			w.WriteByte(0)  // m_nCalcDamageStatIndex
			if stat.IsMovementAffectingStat(t) {
				w.WriteByte(0) // bStat
			}
			return w.Bytes()
		}
	}
}

func MonsterStatResetBody(uniqueId uint32, stat *model.MonsterTemporaryStat) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteByteArray(stat.Encode(l, ctx)(options))
			w.WriteInt16(0) // tDelay
			w.WriteByte(0)  // m_nCalcDamageStatIndex
			if stat.IsMovementAffectingStat(t) {
				w.WriteByte(0) // bStat
			}
			return w.Bytes()
		}
	}
}
