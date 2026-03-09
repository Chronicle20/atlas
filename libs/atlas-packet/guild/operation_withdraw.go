package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Withdraw struct {
	cid  uint32
	name string
}

func (m Withdraw) Cid() uint32  { return m.cid }
func (m Withdraw) Name() string { return m.name }

func (m Withdraw) Operation() string { return "Withdraw" }

func (m Withdraw) String() string {
	return fmt.Sprintf("cid [%d] name [%s]", m.cid, m.name)
}

func (m Withdraw) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *Withdraw) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.name = r.ReadAsciiString()
	}
}
