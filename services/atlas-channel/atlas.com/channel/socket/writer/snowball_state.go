package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SnowballStateBody(state byte, leftSnowmanHp uint32, rightSnowmanHp uint32, snowball0X uint16, snowball0Y byte, snowball1X uint16, snowball1Y byte, first bool, damageSnowBall uint16, damageSnowMan0 uint16, damageSnowMan1 uint16) packet.Encode {
	return fieldcb.NewSnowballState(state, leftSnowmanHp, rightSnowmanHp, snowball0X, snowball0Y, snowball1X, snowball1Y, first, damageSnowBall, damageSnowMan0, damageSnowMan1).Encode
}
