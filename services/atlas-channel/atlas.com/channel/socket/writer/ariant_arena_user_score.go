package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func AriantArenaUserScoreBody(count byte, name string, score uint32) packet.Encode {
	return fieldcb.NewAriantArenaUserScore(count, name, score).Encode
}
