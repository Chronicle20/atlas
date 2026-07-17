package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseItemMegaphone is the USE_CASH_ITEM sub-body for the Item Megaphone
// (5073xxx). Cosmic-derived (UseCashItemHandler case 3); per-version IDA
// verification in task-123 phases 19-20.
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
