package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func WeddingProgressBody(step byte, groomId uint32, brideId uint32) packet.Encode {
	return fieldcb.NewWeddingProgress(step, groomId, brideId).Encode
}
