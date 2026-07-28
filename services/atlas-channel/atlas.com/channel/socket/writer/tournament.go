package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentBody(mode byte, value byte) packet.Encode {
	return fieldcb.NewTournament(mode, value).Encode
}
