package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MobSkillDelayBody encodes the clientbound MOB_SKILL_DELAY packet
// (CMob::OnMobSkillDelay), which schedules a delayed mob skill. No emitter wires
// this writer yet; it is an intentional seam.
func MobSkillDelayBody(delay int32, skillId int32, skillLevel int32, option int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobSkillDelay(delay, skillId, skillLevel, option).Encode(l, ctx)(options)
		}
	}
}
