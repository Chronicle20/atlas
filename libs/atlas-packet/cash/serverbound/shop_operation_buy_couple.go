package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyCoupleHandle = "CashShopOperationBuyCoupleHandle"

// ShopOperationBuyCouple - CCashShop::OnBuyCouple. The leading field is the
// secondary-password gate (ask_SPW): a 4-byte int in v83, a length-prefixed
// string (EncodeStr) in v95.
type ShopOperationBuyCouple struct {
	birthday     uint32 // v83 leading ask_SPW int
	spw          string // v95 leading ask_SPW string
	option       uint32
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationBuyCouple) Birthday() uint32     { return m.birthday }
func (m ShopOperationBuyCouple) SPW() string           { return m.spw }
func (m ShopOperationBuyCouple) Option() uint32       { return m.option }
func (m ShopOperationBuyCouple) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuyCouple) Name() string          { return m.name }
func (m ShopOperationBuyCouple) Message() string       { return m.message }

func (m ShopOperationBuyCouple) Operation() string {
	return CashShopOperationBuyCoupleHandle
}

func (m ShopOperationBuyCouple) String() string {
	return fmt.Sprintf("birthday [%d], spw [%s], option [%d], serialNumber [%d], name [%s], message [%s]", m.birthday, m.spw, m.option, m.serialNumber, m.name, m.message)
}

func (m ShopOperationBuyCouple) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "JMS" {
			m.encodeJMS(w)
		} else {
			m.encodeGMS(t, w)
		}
		return w.Bytes()
	}
}

func (m ShopOperationBuyCouple) encodeGMS(t tenant.Model, w *response.Writer) {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		w.WriteAsciiString(m.spw)
	} else {
		w.WriteInt(m.birthday)
	}
	w.WriteInt(m.option)
	w.WriteInt(m.serialNumber)
	w.WriteAsciiString(m.name)
	w.WriteAsciiString(m.message)
}

// encodeJMS - JMS185 CCashShop::OnBuyCouple@0x48085a (sub-op 0x1E consumed by
// routing): EncodeStr(SPW), Encode4(nCommSN), EncodeStr(sGiveTo recipient),
// EncodeStr(sText message). No birthday, no option int.
func (m ShopOperationBuyCouple) encodeJMS(w *response.Writer) {
	w.WriteAsciiString(m.spw)
	w.WriteInt(m.serialNumber)
	w.WriteAsciiString(m.name)
	w.WriteAsciiString(m.message)
}

func (m *ShopOperationBuyCouple) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "JMS" {
			m.decodeJMS(r)
		} else {
			m.decodeGMS(t, r)
		}
	}
}

func (m *ShopOperationBuyCouple) decodeGMS(t tenant.Model, r *request.Reader) {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		m.spw = r.ReadAsciiString()
	} else {
		m.birthday = r.ReadUint32()
	}
	m.option = r.ReadUint32()
	m.serialNumber = r.ReadUint32()
	m.name = r.ReadAsciiString()
	m.message = r.ReadAsciiString()
}

func (m *ShopOperationBuyCouple) decodeJMS(r *request.Reader) {
	m.spw = r.ReadAsciiString()
	m.serialNumber = r.ReadUint32()
	m.name = r.ReadAsciiString()
	m.message = r.ReadAsciiString()
}
