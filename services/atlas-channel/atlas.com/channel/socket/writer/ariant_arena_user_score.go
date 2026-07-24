package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func AriantArenaUserScoreBody(entries []fieldcb.AriantArenaScoreEntry) packet.Encode {
	return fieldcb.NewAriantArenaUserScore(entries).Encode
}
