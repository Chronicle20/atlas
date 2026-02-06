package handler

import (
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterMovementHandle = "MonsterMovementHandle"

func MonsterMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		uniqueId := r.ReadUint32()
		moveId := r.ReadInt16()
		dwFlag := r.ReadByte()
		nActionAndDir := r.ReadInt8()
		skillData := r.ReadUint32()
		skillId := int16(skillData & 0xFF)
		skillLevel := int16(skillData >> 8 & 0xFF)

		multiTargetForBall := model.MultiTargetForBall{}
		randTimeForAreaAttack := model.RandTimeForAreaAttack{}
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			multiTargetForBall.Decode(l, t, readerOptions)(r)
			randTimeForAreaAttack.Decode(l, t, readerOptions)(r)
		}

		r.ReadByte()   // moveFlags
		r.ReadUint32() // getHackedCode
		r.ReadUint32() // flyCtxTargetX
		r.ReadUint32() // flyCtxTargetY
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			r.ReadUint32() // dwHackedCodeCRC
		}

		mp := model.Movement{}
		mp.Decode(l, t, readerOptions)(r)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			r.ReadByte()   // bChasing
			r.ReadByte()   // hasTarget | pTarget != 0
			r.ReadByte()   // bChasing 2
			r.ReadByte()   // bChasingHack
			r.ReadUint32() // tChaseDuration
		}
		monsterMoveStartResult := dwFlag > 0

		l.Debugf("Monster [%d] moved. MoveId [%d], dwFlag [%d], nActionAndDir [%d], skillData [%d].", uniqueId, moveId, dwFlag, nActionAndDir, skillData)
		_ = movement.NewProcessor(l, ctx, wp).ForMonster(s.Field(), s.CharacterId(), uniqueId, moveId, monsterMoveStartResult, nActionAndDir, skillId, skillLevel, multiTargetForBall, randTimeForAreaAttack, mp)
	}
}
