package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobEscortFullPathBody encodes the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath), which delivers an escort mob's full waypoint path:
// the previous destination point (oldDestX/oldDestY → CVecCtrlMob::m_Old_Dest.dp),
// the waypoint array (the count is derived from len(waypoints)), the current
// destination index, and the two escort-stop flags — a timed stop
// (hasStopDuration + stopDuration) and an indefinite one (stopIndefinitely).
// Routed at gms_v92 (0x128), gms_v95 (0x130) and jms_v185 (0x110). No emitter
// wires this writer yet; it is an intentional seam.
func MobEscortFullPathBody(oldDestX int32, oldDestY int32, waypoints []monsterpkt.MobEscortWaypoint, currentDestIndex int32, hasStopDuration bool, stopDuration int32, stopIndefinitely bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobEscortFullPath(oldDestX, oldDestY, waypoints, currentDestIndex, hasStopDuration, stopDuration, stopIndefinitely).Encode(l, ctx)(options)
		}
	}
}
