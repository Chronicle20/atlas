package writer

import (
	"atlas-channel/character"
	"context"

	"github.com/sirupsen/logrus"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func CharacterAttackRangedBody(c character.Model, ai packetmodel.AttackInfo) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			skillLevel, mastery, bulletItemId := preComputeAttackValues(l, ctx, c, ai)
			isStrafe := skill2.Is(skill2.Id(ai.SkillId()), skill2.SniperStrafeId)
			hasKeydown := isKeydownSkill(ai.SkillId())
			return charpkt.NewAttackRanged(c.Id(), c.Level(), skillLevel, mastery, bulletItemId, isStrafe, hasKeydown, ai).Encode(l, ctx)(options)
		}
	}
}
