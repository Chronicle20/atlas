package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentSortWriter = "CompartmentSortW"

type CompartmentSortW struct {
	inventoryType byte
}

func NewCompartmentSortW(inventoryType byte) CompartmentSortW {
	return CompartmentSortW{inventoryType: inventoryType}
}

func (m CompartmentSortW) InventoryType() byte { return m.inventoryType }
func (m CompartmentSortW) Operation() string   { return CompartmentSortWriter }
func (m CompartmentSortW) String() string {
	return fmt.Sprintf("inventoryType [%d]", m.inventoryType)
}

func (m CompartmentSortW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteByte(m.inventoryType)
		return w.Bytes()
	}
}

func (m *CompartmentSortW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // always 0
		m.inventoryType = r.ReadByte()
	}
}
