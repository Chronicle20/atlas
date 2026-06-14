package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MobEscortReturnBeforeBody encodes the clientbound MOB_ESCORT_RETURN_BEFORE
// packet (CMob::OnEscortReturnBefore), used during escort sequences. v95 + jms.
// No emitter wires this writer yet; it is an intentional seam.
func MobEscortReturnBeforeBody(index int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobEscortReturnBefore(index).Encode(l, ctx)(options)
		}
	}
}
