package handler

import (
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func MonsterMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := monster.Movement{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = movement.NewProcessor(l, ctx, wp).ForMonster(s.Field(), s.CharacterId(), p.UniqueId(), p.MoveId(), p.MonsterMoveStartResult(), p.ActionAndDir(), p.SkillId(), p.SkillLevel(), p.MultiTargetForBall(), p.RandTimeForAreaAttack(), p.MovementData())
	}
}
