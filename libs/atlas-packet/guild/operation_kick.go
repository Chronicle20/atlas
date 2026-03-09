package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Kick struct {
	cid  uint32
	name string
}

func (m Kick) Cid() uint32  { return m.cid }
func (m Kick) Name() string { return m.name }

func (m Kick) Operation() string { return "Kick" }

func (m Kick) String() string {
	return fmt.Sprintf("cid [%d] name [%s]", m.cid, m.name)
}

func (m Kick) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *Kick) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.name = r.ReadAsciiString()
	}
}
