package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func CoconutHitBody(id uint16, action uint16, hits byte) packet.Encode {
	return fieldcb.NewCoconutHit(id, action, hits).Encode
}
