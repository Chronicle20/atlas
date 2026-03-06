package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MoveMonsterAck = "MoveMonsterAck"

func MoveMonsterAckBody(uniqueId uint32, moveId int16, mp uint16, useSkills bool, skillId byte, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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
