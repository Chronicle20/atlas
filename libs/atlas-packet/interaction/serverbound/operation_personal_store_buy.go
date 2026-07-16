package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// packet-audit:fname CPersonalShopDlg::BuyItem
type OperationPersonalStoreBuy struct {
	index    byte
	quantity uint16
	itemCRC  uint32
}

func (m OperationPersonalStoreBuy) Index() byte { return m.index }

func (m OperationPersonalStoreBuy) Quantity() uint16 { return m.quantity }

func (m OperationPersonalStoreBuy) ItemCRC() uint32 { return m.itemCRC }

func (m OperationPersonalStoreBuy) Operation() string { return "OperationPersonalStoreBuy" }

func (m OperationPersonalStoreBuy) String() string {
	return fmt.Sprintf("index [%d], quantity [%d], itemCRC [%d]", m.index, m.quantity, m.itemCRC)
}

func (m OperationPersonalStoreBuy) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.index)
		w.WriteShort(m.quantity)
		if tradeCrcPresent(t) {
			w.WriteInt(m.itemCRC)
		}
		return w.Bytes()
	}
}

func (m *OperationPersonalStoreBuy) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadByte()
		m.quantity = r.ReadUint16()
		if tradeCrcPresent(t) {
			m.itemCRC = r.ReadUint32()
		}
	}
}
