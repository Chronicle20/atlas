package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func TournamentMatchTableBody() packet.Encode {
	return fieldcb.NewTournamentMatchTable().Encode
}
