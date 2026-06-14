package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MobAttackedByMobBody encodes the clientbound MOB_ATTACKED_BY_MOB packet
// (CMob::OnMobAttackedByMob), which reports a mob taking damage from another mob.
// No emitter wires this writer yet; it is an intentional seam.
func MobAttackedByMobBody(attackIndex int8, damage int32, mobTemplateId int32, left bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobAttackedByMob(attackIndex, damage, mobTemplateId, left).Encode(l, ctx)(options)
		}
	}
}
