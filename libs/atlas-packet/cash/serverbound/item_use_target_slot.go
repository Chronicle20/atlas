package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseTargetSlot is the bare-target-slot sub-body of the cash ItemUse
// packet: one int16 equip position (negative when the target is equipped),
// followed by the trailing updateTime on the two builds that carry it there.
//
// The client shares ONE dispatch arm between the Item Tag case (25) and the
// item-expiration-extender / Magical Sandglass case (61 on GMS < 95, 62 on
// GMS >= 95) — gms_v83 CWvsContext::SendConsumeCashItemUseRequest jump-table
// target @0xA0CAE0, which IDA labels "jumptable 00A0A6E6 cases 25,61" and
// which performs exactly one COutPacket::Encode2. The gms_v95 PDB types the
// encoded argument as `unsigned __int16 nEPOS`. The type is therefore named
// for its layout, not for either caller.
type ItemUseTargetSlot struct {
	slot            int16
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseTargetSlot(updateTimeFirst bool) *ItemUseTargetSlot {
	return &ItemUseTargetSlot{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTargetSlot) Slot() int16        { return m.slot }
func (m ItemUseTargetSlot) UpdateTime() uint32 { return m.updateTime }
func (m ItemUseTargetSlot) Operation() string  { return "ItemUseTargetSlot" }

func (m ItemUseTargetSlot) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseTargetSlot) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
