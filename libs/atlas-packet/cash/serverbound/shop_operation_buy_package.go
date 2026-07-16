package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// legacyGMS reports whether the tenant is a pre-v83 GMS build. The v79
// CASHSHOP_OPERATION arms carry a leaner body than v83+ (fields such as
// pointType/option/currency were added at/after v83), so legacy GMS takes a
// reduced encode/decode path. JMS and GMS>=83 keep the modern shape.
func legacyGMS(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 83
}

const CashShopOperationBuyPackageHandle = "CashShopOperationBuyPackageHandle"

// ShopOperationBuyPackage - CCashShop::SendBuyPackage
// packet-audit:fname CCashShop::OnBuyPackage
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

func (m ShopOperationBuyPackage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// v79 CCashShop::OnBuyPackage@0x468a40: COutPacket(221) Encode1(0x20)=mode
		// (routed op), then Encode4(a2)=serialNumber only. No pointType/option.
		if legacyGMS(t) {
			w.WriteInt(m.serialNumber)
			return w.Bytes()
		}
		w.WriteBool(m.pointType)
		w.WriteInt(m.option)
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyPackage) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if legacyGMS(t) {
			m.serialNumber = r.ReadUint32()
			return
		}
		m.pointType = r.ReadBool()
		m.option = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
	}
}
