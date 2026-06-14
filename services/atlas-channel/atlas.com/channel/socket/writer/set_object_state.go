package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SetObjectStateBody(name string, state uint32) packet.Encode {
	return fieldcb.NewSetObjectState(name, state).Encode
}
