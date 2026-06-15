package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func WitchTowerScoreUpdateBody(score byte, seconds uint32) packet.Encode {
	return fieldcb.NewWitchTowerScoreUpdate(score, seconds).Encode
}
