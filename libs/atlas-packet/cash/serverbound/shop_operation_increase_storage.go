package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationIncreaseStorageHandle = "CashShopOperationIncreaseStorageHandle"

// ShopOperationIncreaseStorage - CCashShop::SendIncTrunkCount
type ShopOperationIncreaseStorage struct {
	isPoints     bool
	currency     uint32
	item         bool
	serialNumber uint32
}

func (m ShopOperationIncreaseStorage) IsPoints() bool       { return m.isPoints }
func (m ShopOperationIncreaseStorage) Currency() uint32      { return m.currency }
func (m ShopOperationIncreaseStorage) Item() bool            { return m.item }
func (m ShopOperationIncreaseStorage) SerialNumber() uint32  { return m.serialNumber }

func (m ShopOperationIncreaseStorage) Operation() string {
	return CashShopOperationIncreaseStorageHandle
}

func (m ShopOperationIncreaseStorage) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], item [%t], serialNumber [%d]", m.isPoints, m.currency, m.item, m.serialNumber)
}

func (m ShopOperationIncreaseStorage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		w.WriteInt(m.currency)
		w.WriteBool(m.item)
		if m.item {
			w.WriteInt(m.serialNumber)
		}
		return w.Bytes()
	}
}

func (m *ShopOperationIncreaseStorage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		m.currency = r.ReadUint32()
		m.item = r.ReadBool()
		if m.item {
			m.serialNumber = r.ReadUint32()
		}
	}
}
