package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyHandle = "CashShopOperationBuyHandle"

// ShopOperationBuy - CCashShop::OnBuy
type ShopOperationBuy struct {
	isPoints     bool
	currency     uint32
	serialNumber uint32
	zero         uint32 // v83 trailing IsZeroGoods int
	oneADay      byte   // v87+ trailing m_bRequestBuyOneADay byte
	eventSN      uint32 // v87+ trailing nEventSN int
}

func (m ShopOperationBuy) IsPoints() bool       { return m.isPoints }
func (m ShopOperationBuy) Currency() uint32      { return m.currency }
func (m ShopOperationBuy) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuy) Zero() uint32          { return m.zero }
func (m ShopOperationBuy) OneADay() byte          { return m.oneADay }
func (m ShopOperationBuy) EventSN() uint32        { return m.eventSN }

func (m ShopOperationBuy) Operation() string {
	return CashShopOperationBuyHandle
}

func (m ShopOperationBuy) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], serialNumber [%d], zero [%d], oneADay [%d], eventSN [%d]", m.isPoints, m.currency, m.serialNumber, m.zero, m.oneADay, m.eventSN)
}

func (m ShopOperationBuy) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m ShopOperationBuy) encodeGMS(t tenant.Model, w *response.Writer) {
	w.WriteBool(m.isPoints)
	w.WriteInt(m.currency)
	w.WriteInt(m.serialNumber)
	if t.Region() == "GMS" && t.MajorVersion() >= 87 {
		w.WriteByte(m.oneADay)
		w.WriteInt(m.eventSN)
	} else {
		w.WriteInt(m.zero)
	}
}

// encodeJMS - JMS185 CCashShop::OnBuy@0x47eaa7: Encode1(usePoints),
// Encode4(nCommSN). No currency, no trailing v83/v87 fields.
func (m ShopOperationBuy) encodeJMS(w *response.Writer) {
	w.WriteBool(m.isPoints)
	w.WriteInt(m.serialNumber)
}

func (m *ShopOperationBuy) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "JMS" {
			m.decodeJMS(r)
		} else {
			m.decodeGMS(t, r)
		}
	}
}

func (m *ShopOperationBuy) decodeGMS(t tenant.Model, r *request.Reader) {
	m.isPoints = r.ReadBool()
	m.currency = r.ReadUint32()
	m.serialNumber = r.ReadUint32()
	if t.Region() == "GMS" && t.MajorVersion() >= 87 {
		m.oneADay = r.ReadByte()
		m.eventSN = r.ReadUint32()
	} else {
		m.zero = r.ReadUint32()
	}
}

func (m *ShopOperationBuy) decodeJMS(r *request.Reader) {
	m.isPoints = r.ReadBool()
	m.serialNumber = r.ReadUint32()
}
