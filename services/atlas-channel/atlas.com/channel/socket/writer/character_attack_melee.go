package writer

import (
	"atlas-channel/character"
	"context"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CharacterAttackMeleeBody(c character.Model, ai packetmodel.AttackInfo) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			skillLevel, mastery, bulletItemId := preComputeAttackValues(l, ctx, c, ai)
			isMesoExplosion := skill2.Is(skill2.Id(ai.SkillId()), skill2.ChiefBanditMesoExplosionId)
			hasKeydown := isKeydownSkill(ai.SkillId())
			return charpkt.NewAttackMelee(c.Id(), c.Level(), skillLevel, mastery, bulletItemId, isMesoExplosion, hasKeydown, ai).Encode(l, ctx)(options)
		}
	}
}
