package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func BlockedMapBody(reason byte) packet.Encode {
	return fieldcb.NewBlockedMap(reason).Encode
}
