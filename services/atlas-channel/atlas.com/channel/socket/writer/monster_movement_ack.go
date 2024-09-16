package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MoveMonsterAck = "MoveMonsterAck"

func MoveMonsterAckBody(_ logrus.FieldLogger, _ tenant.Model) func(uniqueId uint32, moveId int16, mp uint16, useSkills bool, skillId byte, skillLevel byte) BodyProducer {
	return func(uniqueId uint32, moveId int16, mp uint16, useSkills bool, skillId byte, skillLevel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteInt16(moveId)
			w.WriteBool(useSkills)
			w.WriteShort(mp)
			w.WriteByte(skillId)
			w.WriteByte(skillLevel)
			return w.Bytes()
		}
	}
}
