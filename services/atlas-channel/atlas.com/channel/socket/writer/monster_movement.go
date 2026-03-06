package writer

import (
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MoveMonster = "MoveMonster"

func MoveMonsterBody(uniqueId uint32, bNotForceLandingWhenDiscard bool, bNotChangeAction bool, bNextAttackPossible bool, bLeft int8, skillId int16, skillLevel int16, multiTargets model.MultiTargetForBall, randTimeForAreaAttack model.RandTimeForAreaAttack, movement model.Movement) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteBool(bNotForceLandingWhenDiscard)
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteBool(bNotChangeAction)
			}
			w.WriteBool(bNextAttackPossible)
			w.WriteInt8(bLeft)
			w.WriteInt16(skillId)
			w.WriteInt16(skillLevel)
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteByteArray(multiTargets.Encoder(l, ctx)(options))
				w.WriteByteArray(randTimeForAreaAttack.Encoder(l, ctx)(options))
			}
			w.WriteByteArray(movement.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}
