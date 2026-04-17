package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyCoupleHandle = "CashShopOperationBuyCoupleHandle"

// ShopOperationBuyCouple - CCashShop::SendBuyCouple
type ShopOperationBuyCouple struct {
	birthday     uint32
	option       uint32
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationBuyCouple) Birthday() uint32     { return m.birthday }
func (m ShopOperationBuyCouple) Option() uint32       { return m.option }
func (m ShopOperationBuyCouple) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuyCouple) Name() string          { return m.name }
func (m ShopOperationBuyCouple) Message() string       { return m.message }

func (m ShopOperationBuyCouple) Operation() string {
	return CashShopOperationBuyCoupleHandle
}

func (m ShopOperationBuyCouple) String() string {
	return fmt.Sprintf("birthday [%d], option [%d], serialNumber [%d], name [%s], message [%s]", m.birthday, m.option, m.serialNumber, m.name, m.message)
}

func (m ShopOperationBuyCouple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.birthday)
		w.WriteInt(m.option)
		w.WriteInt(m.serialNumber)
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyCouple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.birthday = r.ReadUint32()
		m.option = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
		m.name = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
