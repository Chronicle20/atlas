package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func BlockedServerBody(reason byte) packet.Encode {
	return fieldcb.NewBlockedServer(reason).Encode
}
