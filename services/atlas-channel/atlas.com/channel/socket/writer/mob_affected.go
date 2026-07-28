package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobAffectedBody encodes the clientbound MOB_AFFECTED packet, which marks a mob
// as under a skill area-affect for a bounded duration. No emitter wires this
// writer yet; it is an intentional seam (the codec + route exist so the feature
// can be turned on without a follow-up packet-plumbing pass).
func MobAffectedBody(skillId int32, delay uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobAffected(skillId, delay).Encode(l, ctx)(options)
		}
	}
}
