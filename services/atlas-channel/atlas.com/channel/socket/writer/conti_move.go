package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ContiMoveBody(state byte, subState byte) packet.Encode {
	return fieldcb.NewContiMove(state, subState).Encode
}
