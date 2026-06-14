package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func HorntailCaveBody(state byte, seconds uint32) packet.Encode {
	return fieldcb.NewHorntailCave(state, seconds).Encode
}
