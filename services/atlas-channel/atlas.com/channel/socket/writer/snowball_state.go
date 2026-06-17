package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SnowballStateBody(state byte, leftSnow uint32, rightSnow uint32, snowmanHp uint16, position byte, x0 uint16, x1 uint16, x2 uint16) packet.Encode {
	return fieldcb.NewSnowballState(state, leftSnow, rightSnow, snowmanHp, position, x0, x1, x2).Encode
}
