package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SheepRanchInfoBody(wolfCount byte, wolfDisguisedCount byte) packet.Encode {
	return fieldcb.NewSheepRanchInfo(wolfCount, wolfDisguisedCount).Encode
}
