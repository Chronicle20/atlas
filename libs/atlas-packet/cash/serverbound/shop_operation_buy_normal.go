package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopOperationBuyNormalHandle = "CashShopOperationBuyNormalHandle"

// ShopOperationBuyNormal - CCashShop::SendBuyNormal. On v83+ this is a bare
// serialNumber(4) buy-for-self. In v48 the IDB-labeled CCashShop::OnBuyNormal
// @0x44cbb2 is actually a gift-shaped flow (send @0x44cdaf): COutPacket(160)
// Encode1(0x1F)=mode, Encode4(ask_SPW)=spw int, Encode4(a2)=serialNumber,
// EncodeStr(recipient name), EncodeStr(message). The extra spw/name/message
// fields are v48-only (legacy GMS < 61).
// packet-audit:fname CCashShop::OnBuyNormal
type ShopOperationBuyNormal struct {
	serialNumber uint32
	spw          uint32 // v48 leading ask_SPW int
	name         string // v48 recipient name
	message      string // v48 gift message
}

func (m ShopOperationBuyNormal) SerialNumber() uint32 { return m.serialNumber }
func (m ShopOperationBuyNormal) SPW() uint32          { return m.spw }
func (m ShopOperationBuyNormal) Name() string         { return m.name }
func (m ShopOperationBuyNormal) Message() string      { return m.message }

func (m ShopOperationBuyNormal) Operation() string {
	return CashShopOperationBuyNormalHandle
}

func (m ShopOperationBuyNormal) String() string {
	return fmt.Sprintf("serialNumber [%d], spw [%d], name [%s], message [%s]", m.serialNumber, m.spw, m.name, m.message)
}

func (m ShopOperationBuyNormal) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if buyOmitsCurrency(t) {
			w.WriteInt(m.spw)
			w.WriteInt(m.serialNumber)
			w.WriteAsciiString(m.name)
			w.WriteAsciiString(m.message)
			return w.Bytes()
		}
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyNormal) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if buyOmitsCurrency(t) {
			m.spw = r.ReadUint32()
			m.serialNumber = r.ReadUint32()
			m.name = r.ReadAsciiString()
			m.message = r.ReadAsciiString()
			return
		}
		m.serialNumber = r.ReadUint32()
	}
}
