package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationRebateLockerItemHandle = "CashShopOperationRebateLockerItemHandle"

// ShopOperationRebateLockerItem - CCashShop::SendRebateLockerItem
type ShopOperationRebateLockerItem struct {
	birthday uint32
	unk      uint64
}

func (m ShopOperationRebateLockerItem) Birthday() uint32 { return m.birthday }
func (m ShopOperationRebateLockerItem) Unk() uint64      { return m.unk }

func (m ShopOperationRebateLockerItem) Operation() string {
	return CashShopOperationRebateLockerItemHandle
}

func (m ShopOperationRebateLockerItem) String() string {
	return fmt.Sprintf("birthday [%d], unk [%d]", m.birthday, m.unk)
}

func (m ShopOperationRebateLockerItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.birthday)
		w.WriteLong(m.unk)
		return w.Bytes()
	}
}

func (m *ShopOperationRebateLockerItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.birthday = r.ReadUint32()
		m.unk = r.ReadUint64()
	}
}
