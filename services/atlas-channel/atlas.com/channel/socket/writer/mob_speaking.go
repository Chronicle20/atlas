package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobSpeakingBody encodes the clientbound MOB_SPEAKING packet
// (CMob::OnMobSpeaking), which triggers a mob speech/animation pair. No emitter
// wires this writer yet; it is an intentional seam.
func MobSpeakingBody(speechType int32, action int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobSpeaking(speechType, action).Encode(l, ctx)(options)
		}
	}
}
