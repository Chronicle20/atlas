package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const MoveMonsterAck = "MoveMonsterAck"

func MoveMonsterAckBody(uniqueId uint32, moveId int16, mp uint16, useSkills bool, skillId byte, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return monsterpkt.NewMonsterMovementAck(uniqueId, moveId, mp, useSkills, skillId, skillLevel).Encode(l, ctx)
	}
}
