package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyNameChangeHandle = "CashShopOperationBuyNameChangeHandle"

// ShopOperationBuyNameChange - CCashShop::SendBuyNameChange
type ShopOperationBuyNameChange struct {
	serialNumber uint32
	oldName      string
	newName      string
}

func (m ShopOperationBuyNameChange) SerialNumber() uint32 { return m.serialNumber }
func (m ShopOperationBuyNameChange) OldName() string      { return m.oldName }
func (m ShopOperationBuyNameChange) NewName() string      { return m.newName }

func (m ShopOperationBuyNameChange) Operation() string {
	return CashShopOperationBuyNameChangeHandle
}

func (m ShopOperationBuyNameChange) String() string {
	return fmt.Sprintf("serialNumber [%d], oldName [%s], newName [%s]", m.serialNumber, m.oldName, m.newName)
}

func (m ShopOperationBuyNameChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		w.WriteAsciiString(m.oldName)
		w.WriteAsciiString(m.newName)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyNameChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
		m.oldName = r.ReadAsciiString()
		m.newName = r.ReadAsciiString()
	}
}
