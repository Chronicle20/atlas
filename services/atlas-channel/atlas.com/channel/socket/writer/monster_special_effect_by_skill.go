package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MonsterSpecialEffectBySkillBody encodes the clientbound
// MONSTER_SPECIAL_EFFECT_BY_SKILL packet, which plays a skill's special hit
// effect on a mob. The characterId/delay args are only emitted on GMS v95+ (the
// codec gates them on region+version). No emitter wires this writer yet; it is an
// intentional seam (the codec + route exist so the feature can be turned on
// without a follow-up packet-plumbing pass).
func MonsterSpecialEffectBySkillBody(uniqueId uint32, skillId int32, characterId int32, delay uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMonsterSpecialEffectBySkill(uniqueId, skillId, characterId, delay).Encode(l, ctx)(options)
		}
	}
}
