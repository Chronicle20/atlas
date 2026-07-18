package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseVegaScroll — the category-561 (Vega's Spell) sub-body of the
// USE_CASH_ITEM request. The packet is assembled and sent by the CUIVega
// dialog, not by CWvsContext::SendConsumeCashItemUseRequest directly (v83
// sub_82CBE2 LABEL_28; v95 CUIVega::OnButtonClicked 0x7bf4a0). The trailing
// updateTime is present on EVERY version regardless of the prefix's
// updateTimeFirst convention — v95 carries updateTime in the prefix AND here
// (both IDA-verified, task-130 design §2.1) — so Decode reads all six int32s
// unconditionally.
//
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest#VegaScroll
type ItemUseVegaScroll struct {
	equipTab   uint32 // inventory tab index of the equip target; always 1 (equip inventory)
	equipSlot  int32  // positive = equip inventory; sign is passed through to the service
	scrollTab  uint32 // inventory tab index of the scroll; always 2 (use inventory)
	scrollSlot int32
	flag       uint32 // constant 1 on v83+v95 (v95 IDB names it m_nWhiteScrollUse but always writes 1); read and ignored
	updateTime uint32
}

func NewItemUseVegaScroll(equipTab uint32, equipSlot int32, scrollTab uint32, scrollSlot int32, flag uint32, updateTime uint32) *ItemUseVegaScroll {
	return &ItemUseVegaScroll{
		equipTab:   equipTab,
		equipSlot:  equipSlot,
		scrollTab:  scrollTab,
		scrollSlot: scrollSlot,
		flag:       flag,
		updateTime: updateTime,
	}
}

func (m ItemUseVegaScroll) EquipTab() uint32   { return m.equipTab }
func (m ItemUseVegaScroll) EquipSlot() int32   { return m.equipSlot }
func (m ItemUseVegaScroll) ScrollTab() uint32  { return m.scrollTab }
func (m ItemUseVegaScroll) ScrollSlot() int32  { return m.scrollSlot }
func (m ItemUseVegaScroll) Flag() uint32       { return m.flag }
func (m ItemUseVegaScroll) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseVegaScroll) Operation() string { return "ItemUseVegaScroll" }

func (m ItemUseVegaScroll) String() string {
	return fmt.Sprintf("equipTab [%d] equipSlot [%d] scrollTab [%d] scrollSlot [%d] flag [%d] updateTime [%d]",
		m.equipTab, m.equipSlot, m.scrollTab, m.scrollSlot, m.flag, m.updateTime)
}

func (m ItemUseVegaScroll) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.equipTab)
		w.WriteInt32(m.equipSlot)
		w.WriteInt(m.scrollTab)
		w.WriteInt32(m.scrollSlot)
		w.WriteInt(m.flag)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseVegaScroll) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.equipTab = r.ReadUint32()
		m.equipSlot = r.ReadInt32()
		m.scrollTab = r.ReadUint32()
		m.scrollSlot = r.ReadInt32()
		m.flag = r.ReadUint32()
		m.updateTime = r.ReadUint32()
	}
}
