package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentBody(mode byte, arg0 byte, arg1 byte) packet.Encode {
	return fieldcb.NewTournament(mode, arg0, arg1).Encode
}
