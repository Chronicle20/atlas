package handler

import (
	"atlas-channel/character"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const MonsterMovementHandle = "MonsterMovementHandle"

func MonsterMovementHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	moveMonsterAckFunc := session.Announce(l)(wp)(writer.MoveMonsterAck)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		uniqueId := r.ReadUint32()

		m, err := monster.GetById(l, span, s.Tenant())(uniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate monster [%d] moving.", uniqueId)
			return
		}
		c, err := character.GetById(l, span, s.Tenant())(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] issuing the movement.", s.CharacterId())
			return
		}
		if m.WorldId() != s.WorldId() || m.ChannelId() != s.ChannelId() || m.MapId() != c.MapId() {
			l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", m.UniqueId(), s.CharacterId())
			return
		}

		moveId := r.ReadInt16()
		dwFlag := r.ReadByte()
		nActionAndDir := r.ReadByte()
		skillData := r.ReadUint32()
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			nMultiTargetSize := r.ReadUint32()
			for i := 0; i < int(nMultiTargetSize); i++ {
				r.ReadUint32() // x
				r.ReadUint32() // y
			}

			nRandTimeSize := r.ReadUint32()
			for i := 0; i < int(nRandTimeSize); i++ {
				r.ReadUint32()
			}
		}

		r.ReadByte()   // moveFlags
		r.ReadUint32() // getHackedCode
		r.ReadUint32() // flyCtxTargetX
		r.ReadUint32() // flyCtxTargetY
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			r.ReadUint32() // dwHackedCodeCRC
		}

		mp := model.Movement{}
		mp.Decode(l, s.Tenant(), readerOptions)(r)

		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			r.ReadByte()   // bChasing
			r.ReadByte()   // hasTarget | pTarget != 0
			r.ReadByte()   // bChasing 2
			r.ReadByte()   // bChasingHack
			r.ReadUint32() // tChaseDuration
		}

		l.Debugf("Monster [%d] moved. MoveId [%d], dwFlag [%d], nActionAndDir [%d], skillData [%d].", uniqueId, moveId, dwFlag, nActionAndDir, skillData)
		err = moveMonsterAckFunc(s, writer.MoveMonsterAckBody(l, s.Tenant())(uniqueId, moveId, uint16(m.MP()), false, 0, 0))
		if err != nil {
			l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", m.UniqueId(), s.CharacterId())
		}
	}
}
