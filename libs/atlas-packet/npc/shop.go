package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCShopHandle = "NPCShopHandle"

type Shop struct {
	op byte
}

func (m Shop) Op() byte {
	return m.op
}

func (m Shop) Operation() string {
	return NPCShopHandle
}

func (m Shop) String() string {
	return fmt.Sprintf("op [%d]", m.op)
}

func (m Shop) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.op)
		return w.Bytes()
	}
}

func (m *Shop) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.op = r.ReadByte()
	}
}
