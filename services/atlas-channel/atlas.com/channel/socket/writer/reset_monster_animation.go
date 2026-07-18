package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ResetMonsterAnimationBody encodes the clientbound RESET_MONSTER_ANIMATION
// packet, which un-suspends a mob and resets its action layer. No emitter wires
// this writer yet; it is an intentional seam (the codec + route exist so the
// feature can be turned on without a follow-up packet-plumbing pass).
func ResetMonsterAnimationBody(animate bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewResetMonsterAnimation(animate).Encode(l, ctx)(options)
		}
	}
}
