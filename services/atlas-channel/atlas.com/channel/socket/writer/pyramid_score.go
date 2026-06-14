package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func PyramidScoreBody(rank byte, score uint32) packet.Encode {
	return fieldcb.NewPyramidScore(rank, score).Encode
}
