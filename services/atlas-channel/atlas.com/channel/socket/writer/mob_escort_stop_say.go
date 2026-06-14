package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MobEscortStopSayBody encodes the clientbound MOB_ESCORT_STOP_SAY packet
// (CMob::OnEscortStopSay), an escort-stop chat-balloon line. v95 + jms. No emitter
// wires this writer yet; it is an intentional seam.
func MobEscortStopSayBody(duration int32, chatBalloon int32, weather bool, hasText bool, text string, action int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobEscortStopSay(duration, chatBalloon, weather, hasText, text, action).Encode(l, ctx)(options)
		}
	}
}
