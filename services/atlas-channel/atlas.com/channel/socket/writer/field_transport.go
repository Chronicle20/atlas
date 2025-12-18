package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func FieldTransportStateBody(_ logrus.FieldLogger, t tenant.Model) func(state TransportState, overrideAppear bool) BodyProducer {
	return func(state TransportState, overrideAppear bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(state))
			w.WriteBool(overrideAppear)
			return w.Bytes()
		}
	}
}
