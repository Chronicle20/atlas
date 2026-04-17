package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationPersonalStoreAddToBlackList struct {
	slot byte
	name string
}

func (m OperationPersonalStoreAddToBlackList) Slot() byte { return m.slot }

func (m OperationPersonalStoreAddToBlackList) Name() string { return m.name }

func (m OperationPersonalStoreAddToBlackList) Operation() string {
	return "OperationPersonalStoreAddToBlackList"
}

func (m OperationPersonalStoreAddToBlackList) String() string {
	return fmt.Sprintf("slot [%d], name [%s]", m.slot, m.name)
}

func (m OperationPersonalStoreAddToBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.slot)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationPersonalStoreAddToBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
