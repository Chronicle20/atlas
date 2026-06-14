package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SnowballHitBody(position byte, damage uint16, distance uint16) packet.Encode {
	return fieldcb.NewSnowballHit(position, damage, distance).Encode
}
