package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CashShopOperationBuyOtherPackageHandle = "CashShopOperationBuyOtherPackageHandle"

// ShopOperationBuyOtherPackage - CCashShop::OnGiftPackage. Bound only in
// GMS templates (BUY_OTHER_PACKAGE has no JMS mode); derivation.md D3a (§4)
// pins the v95.0 body as spw string, serialNumber uint32, name string,
// message string -- NOT byte-identical to ShopOperationBuyPackage, which
// carries pointType/option instead of spw/name/message. There is no
// `pointType`/`option` field on this send at all: OnGiftPackage uses NX
// Prepaid only (dwOption never varies), so there is nothing for the client
// to encode a payment-method choice into.
// packet-audit:fname CCashShop::OnGiftPackage
type ShopOperationBuyOtherPackage struct {
	spw          string
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationBuyOtherPackage) SPW() string          { return m.spw }
func (m ShopOperationBuyOtherPackage) SerialNumber() uint32 { return m.serialNumber }
func (m ShopOperationBuyOtherPackage) Name() string         { return m.name }
func (m ShopOperationBuyOtherPackage) Message() string      { return m.message }

func (m ShopOperationBuyOtherPackage) Operation() string {
	return CashShopOperationBuyOtherPackageHandle
}

func (m ShopOperationBuyOtherPackage) String() string {
	return fmt.Sprintf("spw [%s], serialNumber [%d], name [%s], message [%s]", m.spw, m.serialNumber, m.name, m.message)
}

func (m ShopOperationBuyOtherPackage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.spw)
		w.WriteInt(m.serialNumber)
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyOtherPackage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.spw = r.ReadAsciiString()
		m.serialNumber = r.ReadUint32()
		m.name = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
