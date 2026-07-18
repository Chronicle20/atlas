package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobEscortFullPathBody encodes the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath), which delivers an escort mob's full waypoint path.
// v95 + jms. No emitter wires this writer yet; it is an intentional seam.
func MobEscortFullPathBody(mode int32, waypoints []monsterpkt.MobEscortWaypoint, tail int32, hasArrive bool, arriveDelay int32, hasReset bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobEscortFullPath(mode, waypoints, tail, hasArrive, arriveDelay, hasReset).Encode(l, ctx)(options)
		}
	}
}
