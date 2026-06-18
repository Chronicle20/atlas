package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func PlayJukeboxBody(itemId int32, playerName string) packet.Encode {
	return fieldcb.NewPlayJukebox(itemId, playerName).Encode
}
