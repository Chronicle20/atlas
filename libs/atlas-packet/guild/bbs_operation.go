package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildBBSHandle = "GuildBBSHandle"

type BBS struct {
	op byte
}

func (m BBS) Op() byte {
	return m.op
}

func (m BBS) Operation() string {
	return "BBS"
}

func (m BBS) String() string {
	return fmt.Sprintf("op [%d]", m.op)
}

func (m BBS) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.op)
		return w.Bytes()
	}
}

func (m *BBS) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.op = r.ReadByte()
	}
}
