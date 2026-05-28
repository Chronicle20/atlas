package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationRebateLockerItemHandle = "CashShopOperationRebateLockerItemHandle"

// ShopOperationRebateLockerItem - CCashShop::OnRebateLockerItem. The leading
// field is the secondary-password gate (ask_SPW): a 4-byte int in v83, a
// length-prefixed string (EncodeStr) in v95. The trailing 8-byte locker serial
// (EncodeBuffer 8) is identical in both versions.
type ShopOperationRebateLockerItem struct {
	birthday uint32 // v83 leading ask_SPW int
	spw      string // v95 leading ask_SPW string
	unk      uint64
}

func (m ShopOperationRebateLockerItem) Birthday() uint32 { return m.birthday }
func (m ShopOperationRebateLockerItem) SPW() string      { return m.spw }
func (m ShopOperationRebateLockerItem) Unk() uint64      { return m.unk }

func (m ShopOperationRebateLockerItem) Operation() string {
	return CashShopOperationRebateLockerItemHandle
}

func (m ShopOperationRebateLockerItem) String() string {
	return fmt.Sprintf("birthday [%d], spw [%s], unk [%d]", m.birthday, m.spw, m.unk)
}

func (m ShopOperationRebateLockerItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteAsciiString(m.spw)
		} else {
			w.WriteInt(m.birthday)
		}
		w.WriteLong(m.unk)
		return w.Bytes()
	}
}

func (m *ShopOperationRebateLockerItem) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.spw = r.ReadAsciiString()
		} else {
			m.birthday = r.ReadUint32()
		}
		m.unk = r.ReadUint64()
	}
}
