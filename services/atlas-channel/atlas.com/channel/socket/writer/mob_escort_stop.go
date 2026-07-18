package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobEscortStopBody encodes the clientbound MOB_ESCORT_STOP packet
// (CMob::OnEscortStopEndPermmision), an empty-payload escort-stop end. v95. No
// emitter wires this writer yet; it is an intentional seam.
func MobEscortStopBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.MobEscortStop{}.Encode(l, ctx)(options)
		}
	}
}
