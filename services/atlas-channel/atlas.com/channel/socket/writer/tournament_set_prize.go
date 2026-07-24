package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentSetPrizeBody(slot byte, flag byte, itemId1 uint32, itemId2 uint32) packet.Encode {
	return fieldcb.NewTournamentSetPrize(slot, flag, itemId1, itemId2).Encode
}
