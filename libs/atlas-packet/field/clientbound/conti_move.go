package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ContiMoveWriter = "ContiMove"

// packet-audit:fname CField_ContiMove::OnContiMove
type ContiMove struct {
	state byte
}

func NewContiMove(state byte) ContiMove {
	return ContiMove{state: state}
}

func (m ContiMove) State() byte { return m.state }

func (m ContiMove) Operation() string { return ContiMoveWriter }
func (m ContiMove) String() string {
	return fmt.Sprintf("state [%d]", m.state)
}

func (m ContiMove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		return w.Bytes()
	}
}

func (m *ContiMove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
	}
}
