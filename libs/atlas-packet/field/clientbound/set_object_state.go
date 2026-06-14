package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetObjectStateWriter = "SetObjectState"

type SetObjectState struct {
	name  string
	state uint32
}

func NewSetObjectState(name string, state uint32) SetObjectState {
	return SetObjectState{name: name, state: state}
}

func (m SetObjectState) Name() string  { return m.name }
func (m SetObjectState) State() uint32 { return m.state }

func (m SetObjectState) Operation() string { return SetObjectStateWriter }
func (m SetObjectState) String() string {
	return fmt.Sprintf("name [%s] state [%d]", m.name, m.state)
}

func (m SetObjectState) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteInt(m.state)
		return w.Bytes()
	}
}

func (m *SetObjectState) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.state = r.ReadUint32()
	}
}
