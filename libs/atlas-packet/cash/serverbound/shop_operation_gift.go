package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationGiftHandle = "CashShopOperationGiftHandle"

// ShopOperationGift - CCashShop::SendGift
type ShopOperationGift struct {
	birthday     uint32
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationGift) Birthday() uint32     { return m.birthday }
func (m ShopOperationGift) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationGift) Name() string          { return m.name }
func (m ShopOperationGift) Message() string       { return m.message }

func (m ShopOperationGift) Operation() string {
	return CashShopOperationGiftHandle
}

func (m ShopOperationGift) String() string {
	return fmt.Sprintf("birthday [%d], serialNumber [%d], name [%s], message [%s]", m.birthday, m.serialNumber, m.name, m.message)
}

func (m ShopOperationGift) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.birthday)
		w.WriteInt(m.serialNumber)
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ShopOperationGift) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.birthday = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
		m.name = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
