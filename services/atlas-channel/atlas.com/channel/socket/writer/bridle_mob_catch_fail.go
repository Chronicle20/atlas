package writer

import (
	"context"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// BridleMobCatchFailBody encodes the clientbound BRIDLE_MOB_CATCH_FAIL packet,
// which notifies the client that a bridle (taming-item) capture attempt failed.
// No emitter wires this writer yet; it is an intentional seam (the codec + route
// exist so the feature can be turned on without a follow-up packet-plumbing pass).
func BridleMobCatchFailBody(reason byte, itemId int32, unused int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return charpkt.NewBridleMobCatchFail(reason, itemId, unused).Encode(l, ctx)(options)
		}
	}
}
