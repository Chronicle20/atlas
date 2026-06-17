package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func MtsOperation2Body(cash uint32, maplePoints uint32) packet.Encode {
	return fieldcb.NewMtsOperation2(cash, maplePoints).Encode
}
