package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func GmEventInstructionsBody(index byte) packet.Encode {
	return fieldcb.NewGmEventInstructions(index).Encode
}
