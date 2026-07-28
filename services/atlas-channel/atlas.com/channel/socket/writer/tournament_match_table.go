package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentMatchTableBody(match [fieldcb.TournamentMatchTableBufferSize]byte, state byte) packet.Encode {
	return fieldcb.NewTournamentMatchTable(match, state).Encode
}
