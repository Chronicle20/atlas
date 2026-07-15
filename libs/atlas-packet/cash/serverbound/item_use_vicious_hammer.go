package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseViciousHammer is the trailing body the CUIItemUpgrade dialog appends
// to the pre-built CASH_ITEM_USE packet when the Upgrade button is clicked
// (v83 CUIItemUpgrade::OnButtonClicked sub_82AED3 / v95 0x7c0ca0):
// Encode4(m_nItemTI) + Encode4(m_nSlotPosition) + Encode4(update_time).
// slotPosition is signed: negative = an equipped item, positive = a slot in
// the equip inventory. Layout is version-invariant (IDA v83 + v95).
type ItemUseViciousHammer struct {
	itemTI       uint32
	slotPosition int32
	updateTime   uint32
}

func NewItemUseViciousHammer() *ItemUseViciousHammer {
	return &ItemUseViciousHammer{}
}

func (m ItemUseViciousHammer) ItemTI() uint32      { return m.itemTI }
func (m ItemUseViciousHammer) SlotPosition() int32 { return m.slotPosition }
func (m ItemUseViciousHammer) UpdateTime() uint32  { return m.updateTime }

func (m ItemUseViciousHammer) Operation() string { return "ItemUseViciousHammer" }

func (m ItemUseViciousHammer) String() string {
	return fmt.Sprintf("itemTI [%d] slotPosition [%d] updateTime [%d]", m.itemTI, m.slotPosition, m.updateTime)
}

func (m ItemUseViciousHammer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemTI)
		w.WriteInt32(m.slotPosition)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseViciousHammer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemTI = r.ReadUint32()
		m.slotPosition = r.ReadInt32()
		m.updateTime = r.ReadUint32()
	}
}
