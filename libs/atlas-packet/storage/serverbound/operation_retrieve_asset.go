package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationRetrieveAsset struct {
	inventoryType byte
	slot          byte
}

func (m OperationRetrieveAsset) InventoryType() byte { return m.inventoryType }
func (m OperationRetrieveAsset) Slot() byte          { return m.slot }

func (m OperationRetrieveAsset) Operation() string { return "OperationRetrieveAsset" }

func (m OperationRetrieveAsset) String() string {
	return fmt.Sprintf("inventoryType [%d] slot [%d]", m.inventoryType, m.slot)
}

func (m OperationRetrieveAsset) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.inventoryType)
		w.WriteByte(m.slot)
		return w.Bytes()
	}
}

func (m *OperationRetrieveAsset) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadByte()
	}
}
