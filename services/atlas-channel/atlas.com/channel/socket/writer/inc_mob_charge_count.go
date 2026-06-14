package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// IncMobChargeCountBody encodes the clientbound INC_MOB_CHARGE_COUNT packet
// (CMob::OnIncMobChargeCount), which updates a mob's charge counter and
// attack-ready flag. No emitter wires this writer yet; it is an intentional seam.
func IncMobChargeCountBody(chargeCount int32, attackReady int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewIncMobChargeCount(chargeCount, attackReady).Encode(l, ctx)(options)
		}
	}
}
