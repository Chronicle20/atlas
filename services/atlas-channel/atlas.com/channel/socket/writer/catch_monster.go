package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// CatchMonsterBody encodes the clientbound CATCH_MONSTER packet, which plays a
// mob-capture effect on a targeted mob. No emitter wires this writer yet; it is
// an intentional seam (the codec + route exist so the feature can be turned on
// without a follow-up packet-plumbing pass).
func CatchMonsterBody(result byte, success byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewCatchMonster(result, success).Encode(l, ctx)(options)
		}
	}
}
