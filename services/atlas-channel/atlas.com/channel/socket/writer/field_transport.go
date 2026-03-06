package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
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
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(state))
			w.WriteBool(overrideAppear)
			return w.Bytes()
		}
	}
}
