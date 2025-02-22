package handler

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterMovementHandle = "MonsterMovementHandle"

func MonsterMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		uniqueId := r.ReadUint32()

		m, err := monster.GetById(l)(ctx)(uniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate monster [%d] moving.", uniqueId)
			return
		}

		if m.WorldId() != s.WorldId() || m.ChannelId() != s.ChannelId() || m.MapId() != s.MapId() {
			l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", m.UniqueId(), s.CharacterId())
			return
		}

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

		l.Debugf("Monster [%d] moved. MoveId [%d], dwFlag [%d], nActionAndDir [%d], skillData [%d].", uniqueId, moveId, dwFlag, nActionAndDir, skillData)
		err = session.Announce(l)(ctx)(wp)(writer.MoveMonsterAck)(writer.MoveMonsterAckBody(l, t)(uniqueId, moveId, uint16(m.MP()), false, 0, 0))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", m.UniqueId(), s.CharacterId())
		}

		monsterMoveStartResult := dwFlag > 0

		err = producer.ProviderImpl(l)(ctx)(monster.EnvCommandMovement)(monster.Move(s.Map(), m.UniqueId(), s.CharacterId(), monsterMoveStartResult, nActionAndDir, skillId, skillLevel, multiTargetForBall, randTimeForAreaAttack, mp))
		if err != nil {
			l.WithError(err).Errorf("Unable to distribute monster movement to other services.")
		}
	}
}
