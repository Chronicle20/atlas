package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobNextAttackBody encodes the clientbound MOB_NEXT_ATTACK packet
// (CMob::OnNextAttack), which tells a mob to evaluate its next attack. v95-only.
// No emitter wires this writer yet; it is an intentional seam.
func MobNextAttackBody(attackId int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobNextAttack(attackId).Encode(l, ctx)(options)
		}
	}
}
