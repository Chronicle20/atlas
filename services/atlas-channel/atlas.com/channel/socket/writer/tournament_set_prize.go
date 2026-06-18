package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentSetPrizeBody(slot byte, flag byte, itemId uint32, count uint32) packet.Encode {
	return fieldcb.NewTournamentSetPrize(slot, flag, itemId, count).Encode
}
