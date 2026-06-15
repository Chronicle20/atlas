package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func AriantScoreBody(score byte) packet.Encode {
	return fieldcb.NewAriantScore(score).Encode
}
