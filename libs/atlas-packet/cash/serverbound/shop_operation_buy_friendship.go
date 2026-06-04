package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyFriendshipHandle = "CashShopOperationBuyFriendshipHandle"

// ShopOperationBuyFriendship - CCashShop::OnBuyFriendship. The leading field is
// the secondary-password gate (ask_SPW): a 4-byte int in v83, a length-prefixed
// string (EncodeStr) in v95.
type ShopOperationBuyFriendship struct {
	birthday     uint32 // v83 leading ask_SPW int
	spw          string // v95 leading ask_SPW string
	option       uint32
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationBuyFriendship) Birthday() uint32     { return m.birthday }
func (m ShopOperationBuyFriendship) SPW() string           { return m.spw }
func (m ShopOperationBuyFriendship) Option() uint32       { return m.option }
func (m ShopOperationBuyFriendship) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuyFriendship) Name() string          { return m.name }
func (m ShopOperationBuyFriendship) Message() string       { return m.message }

func (m ShopOperationBuyFriendship) Operation() string {
	return CashShopOperationBuyFriendshipHandle
}

func (m ShopOperationBuyFriendship) String() string {
	return fmt.Sprintf("birthday [%d], spw [%s], option [%d], serialNumber [%d], name [%s], message [%s]", m.birthday, m.spw, m.option, m.serialNumber, m.name, m.message)
}

func (m ShopOperationBuyFriendship) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m ShopOperationBuyFriendship) encodeGMS(t tenant.Model, w *response.Writer) {
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

// encodeJMS - JMS185 CCashShop::OnBuyFriendship@0x481184 (sub-op 0x24 consumed by
// routing): EncodeStr(SPW), Encode4(nCommSN), EncodeStr(recipient name),
// EncodeStr(message). No birthday, no option int.
func (m ShopOperationBuyFriendship) encodeJMS(w *response.Writer) {
	w.WriteAsciiString(m.spw)
	w.WriteInt(m.serialNumber)
	w.WriteAsciiString(m.name)
	w.WriteAsciiString(m.message)
}

func (m *ShopOperationBuyFriendship) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "JMS" {
			m.decodeJMS(r)
		} else {
			m.decodeGMS(t, r)
		}
	}
}

func (m *ShopOperationBuyFriendship) decodeGMS(t tenant.Model, r *request.Reader) {
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

func (m *ShopOperationBuyFriendship) decodeJMS(r *request.Reader) {
	m.spw = r.ReadAsciiString()
	m.serialNumber = r.ReadUint32()
	m.name = r.ReadAsciiString()
	m.message = r.ReadAsciiString()
}
