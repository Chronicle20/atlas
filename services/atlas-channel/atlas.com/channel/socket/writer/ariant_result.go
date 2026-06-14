package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func AriantResultBody(message string) packet.Encode {
	return fieldcb.NewAriantResult(message).Encode
}
