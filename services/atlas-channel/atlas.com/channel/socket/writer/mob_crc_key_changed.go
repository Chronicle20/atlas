package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MobCrcKeyChangedBody encodes the clientbound MOB_CRC_KEY_CHANGED packet, which
// pushes a refreshed mob-CRC key to the client. No emitter wires this writer yet;
// it is an intentional seam (the codec + route exist so the feature can be turned
// on without a follow-up packet-plumbing pass).
func MobCrcKeyChangedBody(crcKey uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobCrcKeyChanged(crcKey).Encode(l, ctx)(options)
		}
	}
}
