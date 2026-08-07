package writer

import (
	"atlas-channel/character"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func CharacterAttackMeleeBody(c character.Model, ai packetmodel.AttackInfo) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			skillLevel, mastery, bulletItemId := preComputeAttackValues(l, ctx, c, ai)
			// Routed through the resolved Identity (task-187) rather than a
			// raw wire compare, per the MesoExplosion defensive-consistency
			// scope (skill/handler/character_attack_common.go migrates the
			// same skill's validation gate the same way).
			t := tenant.MustFromContext(ctx)
			set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
			attackId, attackIdOk := set.Skill.Resolve(skill2.Id(ai.SkillId()))
			isMesoExplosion := attackIdOk && skill2.IsIdentity(attackId, skill2.ChiefBanditMesoExplosion)
			hasKeydown := isKeydownSkill(ai.SkillId())
			return charpkt.NewAttackMelee(c.Id(), c.Level(), skillLevel, mastery, bulletItemId, isMesoExplosion, hasKeydown, ai).Encode(l, ctx)(options)
		}
	}
}
