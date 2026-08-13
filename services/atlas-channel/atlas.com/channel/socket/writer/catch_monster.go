package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// CatchMonsterBody encodes the clientbound CATCH_MONSTER packet, which plays the
// generic capture image out of Effect/BasicEff.img (CMob::OnCatchEffect ->
// CAnimationDisplayer::Effect_Catch @v83 0x438eb6).
//
// No emitter wires this writer: the item-catch success path deliberately sends
// only CATCH_MONSTER_WITH_ITEM, whose item-keyed animation is the observed
// client render (see handleStatusEventCaught in kafka/consumer/monster). The
// codec + template routes stay as an intentional seam so another mechanic — or
// a reversal of that choice — needs no packet-plumbing pass.
func CatchMonsterBody(uniqueId uint32, result byte, success byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewCatchMonster(uniqueId, result, success).Encode(l, ctx)(options)
		}
	}
}
