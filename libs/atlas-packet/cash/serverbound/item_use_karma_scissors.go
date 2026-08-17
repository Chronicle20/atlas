package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseKarmaScissors is the Scissors of Karma (classification 552) sub-body of
// the cash ItemUse packet, sent by CUIKarmaDlg::_SendConsumeCashItemUseRequest.
//
// Derived per version, both ends of the supported range:
//
//	gms_v83 @0x830FB5, opcode 0x4F:
//	  Encode2(m_nPOS) Encode4(m_nItemID) Encode4(m_nTargetTI) Encode4(m_nTargetPOS) Encode4(get_update_time())
//	gms_v95 @0x7D7EF0, opcode 0x55:
//	  Encode4(get_update_time()) Encode2(m_nPOS) Encode4(m_nItemID) Encode4(m_nTargetTI) Encode4(m_nTargetPOS)
//
// The leading Encode2+Encode4 pair is the common ItemUse header (item_use.go);
// the update_time position difference is the existing UpdateTimeFirst gate. What
// remains here is nTargetTI + nTargetPOS, byte-identical in shape to ItemUseSeal.
//
// This is a DISCRETE struct rather than an alias of ItemUseSeal, per the
// discrete-struct-per-mode rule in docs/packets/DISPATCHER_FAMILY.md. It is
// emphatically NOT ItemUseTargetSlot (a bare int16), which is the Item Tag /
// expiration-extender shape and carries no target inventory type.
type ItemUseKarmaScissors struct {
	inventoryType   int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseKarmaScissors(updateTimeFirst bool) *ItemUseKarmaScissors {
	return &ItemUseKarmaScissors{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseKarmaScissors) InventoryType() int32 { return m.inventoryType }
func (m ItemUseKarmaScissors) Slot() int32          { return m.slot }
func (m ItemUseKarmaScissors) UpdateTime() uint32   { return m.updateTime }
func (m ItemUseKarmaScissors) Operation() string    { return "ItemUseKarmaScissors" }

func (m ItemUseKarmaScissors) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.inventoryType)
		w.WriteInt32(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseKarmaScissors) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadInt32()
		m.slot = r.ReadInt32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
