package writer

import (
	"context"

	"github.com/Chronicle20/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const MoveMonster = "MoveMonster"

func MoveMonsterBody(uniqueId uint32, bNotForceLandingWhenDiscard bool, bNotChangeAction bool, bNextAttackPossible bool, bLeft int8, skillId int16, skillLevel int16, multiTargets model.MultiTargetForBall, randTimeForAreaAttack model.RandTimeForAreaAttack, movement model.Movement) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return monsterpkt.NewMonsterMovementW(uniqueId, bNotForceLandingWhenDiscard, bNotChangeAction, bNextAttackPossible, bLeft, skillId, skillLevel, multiTargets, randTimeForAreaAttack, movement).Encode(l, ctx)
	}
}
