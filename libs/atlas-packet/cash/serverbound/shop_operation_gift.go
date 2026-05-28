package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationGiftHandle = "CashShopOperationGiftHandle"

// ShopOperationGift - CCashShop::SendGiftsPacket. v83 sends two leading ints
// (Encode4 + Encode4 serialNumber) then EncodeStr name, EncodeStr message. The
// byte oneADay (m_bRequestBuyOneADay) inserted between serialNumber and name
// appears from v87 onward (v87 SendGiftsPacket@0x47a168 still sends the leading
// int, NOT the SPW string — only Encode1 oneADay before name). The leading int
// is replaced by EncodeStr sSPW only at v95+.
type ShopOperationGift struct {
	birthday     uint32 // v83/v87 leading int (replaced by spw string in v95+)
	spw          string // v95+ leading ask_SPW string
	serialNumber uint32
	oneADay      byte // v87+ byte inserted before name
	name         string
	message      string
}

func (m ShopOperationGift) Birthday() uint32     { return m.birthday }
func (m ShopOperationGift) SPW() string           { return m.spw }
func (m ShopOperationGift) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationGift) OneADay() byte          { return m.oneADay }
func (m ShopOperationGift) Name() string          { return m.name }
func (m ShopOperationGift) Message() string       { return m.message }

func (m ShopOperationGift) Operation() string {
	return CashShopOperationGiftHandle
}

func (m ShopOperationGift) String() string {
	return fmt.Sprintf("birthday [%d], spw [%s], serialNumber [%d], oneADay [%d], name [%s], message [%s]", m.birthday, m.spw, m.serialNumber, m.oneADay, m.name, m.message)
}

func (m ShopOperationGift) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteAsciiString(m.spw)
		} else {
			w.WriteInt(m.birthday)
		}
		w.WriteInt(m.serialNumber)
		if t.Region() == "GMS" && t.MajorVersion() >= 87 {
			w.WriteByte(m.oneADay)
		}
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ShopOperationGift) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.spw = r.ReadAsciiString()
		} else {
			m.birthday = r.ReadUint32()
		}
		m.serialNumber = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() >= 87 {
			m.oneADay = r.ReadByte()
		}
		m.name = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
