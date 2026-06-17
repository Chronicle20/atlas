package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ZakumShrineBody(state byte, seconds uint32) packet.Encode {
	return fieldcb.NewZakumShrine(state, seconds).Encode
}
