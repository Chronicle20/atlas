package writer

import (
	"atlas-channel/character"
	"context"

	charpkt "github.com/Chronicle20/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CharacterAttackEnergyBody(c character.Model, ai packetmodel.AttackInfo) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			skillLevel, mastery, bulletItemId := preComputeAttackValues(l, ctx, c, ai)
			hasKeydown := isKeydownSkill(ai.SkillId())
			return charpkt.NewAttackEnergy(c.Id(), c.Level(), skillLevel, mastery, bulletItemId, hasKeydown, ai).Encode(l, ctx)(options)
		}
	}
}
