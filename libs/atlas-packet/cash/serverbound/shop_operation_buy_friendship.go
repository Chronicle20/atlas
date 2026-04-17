package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyFriendshipHandle = "CashShopOperationBuyFriendshipHandle"

// ShopOperationBuyFriendship - CCashShop::SendBuyFriendship
type ShopOperationBuyFriendship struct {
	birthday     uint32
	option       uint32
	serialNumber uint32
	name         string
	message      string
}

func (m ShopOperationBuyFriendship) Birthday() uint32     { return m.birthday }
func (m ShopOperationBuyFriendship) Option() uint32       { return m.option }
func (m ShopOperationBuyFriendship) SerialNumber() uint32  { return m.serialNumber }
func (m ShopOperationBuyFriendship) Name() string          { return m.name }
func (m ShopOperationBuyFriendship) Message() string       { return m.message }

func (m ShopOperationBuyFriendship) Operation() string {
	return CashShopOperationBuyFriendshipHandle
}

func (m ShopOperationBuyFriendship) String() string {
	return fmt.Sprintf("birthday [%d], option [%d], serialNumber [%d], name [%s], message [%s]", m.birthday, m.option, m.serialNumber, m.name, m.message)
}

func (m ShopOperationBuyFriendship) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *ShopOperationBuyFriendship) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.birthday = r.ReadUint32()
		m.option = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
		m.name = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
