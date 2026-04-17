package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentSortWriter = "CompartmentSort"

type CompartmentSort struct {
	inventoryType byte
}

func NewCompartmentSort(inventoryType byte) CompartmentSort {
	return CompartmentSort{inventoryType: inventoryType}
}

func (m CompartmentSort) InventoryType() byte { return m.inventoryType }
func (m CompartmentSort) Operation() string   { return CompartmentSortWriter }
func (m CompartmentSort) String() string {
	return fmt.Sprintf("inventoryType [%d]", m.inventoryType)
}

func (m CompartmentSort) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteByte(m.inventoryType)
		return w.Bytes()
	}
}

func (m *CompartmentSort) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // always 0
		m.inventoryType = r.ReadByte()
	}
}
