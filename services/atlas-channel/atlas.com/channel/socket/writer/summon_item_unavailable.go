package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SummonItemUnavailableBody(message byte) packet.Encode {
	return fieldcb.NewSummonItemUnavailable(message).Encode
}
