package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentMergeWriter = "CompartmentMerge"

type CompartmentMerge struct {
	inventoryType byte
}

func NewCompartmentMerge(inventoryType byte) CompartmentMerge {
	return CompartmentMerge{inventoryType: inventoryType}
}

func (m CompartmentMerge) InventoryType() byte { return m.inventoryType }
func (m CompartmentMerge) Operation() string   { return CompartmentMergeWriter }
func (m CompartmentMerge) String() string {
	return fmt.Sprintf("inventoryType [%d]", m.inventoryType)
}

func (m CompartmentMerge) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteByte(m.inventoryType)
		return w.Bytes()
	}
}

func (m *CompartmentMerge) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // always 0
		m.inventoryType = r.ReadByte()
	}
}
