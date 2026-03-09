package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyPackageHandle = "CashShopOperationBuyPackageHandle"

// ShopOperationBuyPackage - CCashShop::SendBuyPackage
type ShopOperationBuyPackage struct {
	pointType    bool
	option       uint32
	serialNumber uint32
}

func (m ShopOperationBuyPackage) PointType() bool       { return m.pointType }
func (m ShopOperationBuyPackage) Option() uint32         { return m.option }
func (m ShopOperationBuyPackage) SerialNumber() uint32   { return m.serialNumber }

func (m ShopOperationBuyPackage) Operation() string {
	return CashShopOperationBuyPackageHandle
}

func (m ShopOperationBuyPackage) String() string {
	return fmt.Sprintf("pointType [%t], option [%d], serialNumber [%d]", m.pointType, m.option, m.serialNumber)
}

func (m ShopOperationBuyPackage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.pointType)
		w.WriteInt(m.option)
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyPackage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.pointType = r.ReadBool()
		m.option = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
	}
}
