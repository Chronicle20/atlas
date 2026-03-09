package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyHandle = "CashShopOperationBuyHandle"

// ShopOperationBuy - CCashShop::SendBuy
type ShopOperationBuy struct {
	isPoints     bool
	currency     uint32
	serialNumber uint32
	zero         uint32
}

func (m ShopOperationBuy) IsPoints() bool       { return m.isPoints }
func (m ShopOperationBuy) Currency() uint32      { return m.currency }
func (m ShopOperationBuy) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuy) Zero() uint32          { return m.zero }

func (m ShopOperationBuy) Operation() string {
	return CashShopOperationBuyHandle
}

func (m ShopOperationBuy) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], serialNumber [%d], zero [%d]", m.isPoints, m.currency, m.serialNumber, m.zero)
}

func (m ShopOperationBuy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		w.WriteInt(m.currency)
		w.WriteInt(m.serialNumber)
		w.WriteInt(m.zero)
		return w.Bytes()
	}
}

func (m *ShopOperationBuy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		m.currency = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
		m.zero = r.ReadUint32()
	}
}
