package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseItemMegaphone is the USE_CASH_ITEM sub-body for the Item Megaphone
// (5076xxx, cash-slot type 14 — IDA-verified via get_cashslot_item_type on
// gms_v95: itemId%10000/1000==6 -> type 14; the Cosmic-derived "5073xxx"
// guess in earlier revisions of this comment did not match: 5073xxx (Heart
// Megaphone) is cash-slot type 0, no send path). The real sender is the
// CItemSpeakerDlg dialog's own OK-button handler, NOT the main
// SendConsumeCashItemUseRequest dispatcher (which only constructs/shows the
// dialog for this cash-slot type).
// packet-audit:fname CItemSpeakerDlg::_SendConsumeCashItemUseRequest
type ItemUseItemMegaphone struct {
	message         string
	whisper         bool
	hasItem         bool
	invType         int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseItemMegaphone(updateTimeFirst bool) *ItemUseItemMegaphone {
	return &ItemUseItemMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseItemMegaphone) Message() string    { return m.message }
func (m ItemUseItemMegaphone) Whisper() bool      { return m.whisper }
func (m ItemUseItemMegaphone) HasItem() bool      { return m.hasItem }
func (m ItemUseItemMegaphone) InvType() int32     { return m.invType }
func (m ItemUseItemMegaphone) Slot() int32        { return m.slot }
func (m ItemUseItemMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseItemMegaphone) Operation() string { return "ItemUseItemMegaphone" }

func (m ItemUseItemMegaphone) String() string {
	return fmt.Sprintf("message [%s] whisper [%t] hasItem [%t] invType [%d] slot [%d] updateTime [%d]",
		m.message, m.whisper, m.hasItem, m.invType, m.slot, m.updateTime)
}

func (m ItemUseItemMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		w.WriteBool(m.whisper)
		w.WriteBool(m.hasItem)
		if m.hasItem {
			w.WriteInt32(m.invType)
			w.WriteInt32(m.slot)
		}
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseItemMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		m.whisper = r.ReadBool()
		m.hasItem = r.ReadBool()
		if m.hasItem {
			m.invType = r.ReadInt32()
			m.slot = r.ReadInt32()
		}
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
