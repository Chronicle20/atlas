package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationPersonalStoreSetVisitor struct {
	slot byte
	name string
}

func (m OperationPersonalStoreSetVisitor) Slot() byte { return m.slot }

func (m OperationPersonalStoreSetVisitor) Name() string { return m.name }

func (m OperationPersonalStoreSetVisitor) Operation() string {
	return "OperationPersonalStoreSetVisitor"
}

func (m OperationPersonalStoreSetVisitor) String() string {
	return fmt.Sprintf("slot [%d], name [%s]", m.slot, m.name)
}

func (m OperationPersonalStoreSetVisitor) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.slot)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationPersonalStoreSetVisitor) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
