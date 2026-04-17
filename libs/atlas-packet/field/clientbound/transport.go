package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldTransportStateWriter = "FieldTransportState"

type TransportState byte

const (
	TransportStateEnter1  TransportState = 0
	TransportStateEnter2  TransportState = 1
	TransportStateMove1   TransportState = 2
	TransportStateAppear1 TransportState = 3
	TransportStateAppear2 TransportState = 4
	TransportStateMove2   TransportState = 5
	TransportStateEnter3  TransportState = 6
)

type Transport struct {
	state          TransportState
	overrideAppear bool
}

func NewFieldTransport(state TransportState, overrideAppear bool) Transport {
	return Transport{state: state, overrideAppear: overrideAppear}
}

func (m Transport) State() TransportState { return m.state }
func (m Transport) OverrideAppear() bool  { return m.overrideAppear }
func (m Transport) Operation() string     { return FieldTransportStateWriter }
func (m Transport) String() string {
	return fmt.Sprintf("state [%d], overrideAppear [%t]", m.state, m.overrideAppear)
}

func (m Transport) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.state))
		w.WriteBool(m.overrideAppear)
		return w.Bytes()
	}
}

func (m *Transport) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = TransportState(r.ReadByte())
		m.overrideAppear = r.ReadBool()
	}
}
