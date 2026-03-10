package writer

import (
	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
)

type TransportState byte

const (
	FieldTransportState = "FieldTransportState"

	TransportStateEnter1  = TransportState(0)
	TransportStateEnter2  = TransportState(1)
	TransportStateMove1   = TransportState(2)
	TransportStateAppear1 = TransportState(3)
	TransportStateAppear2 = TransportState(4)
	TransportStateMove2   = TransportState(5)
	TransportStateEnter3  = TransportState(6)
)

func FieldTransportStateBody(state TransportState, overrideAppear bool) packet.Encode {
	return fieldpkt.NewFieldTransport(fieldpkt.TransportState(state), overrideAppear).Encode
}
