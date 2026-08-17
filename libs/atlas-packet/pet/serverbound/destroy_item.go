package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const PetDestroyItemHandle = "PetDestroyItemHandle"

// DestroyItem is DESTROY_PET_ITEM_REQUEST — the second packet sent by
// CWvsContext::SendActivatePetRequest.
//
// Double-clicking a pet in the CASH tab reaches one function with two arms
// (GMS v95 @0x9f6980, named; GMS v83 @0xa240a2, unnamed but identical):
//
//	if GW_ItemSlotPet::IsDead(item) {
//	    if !CItemInfo::IsNoRevive(itemId) -> Notice "The time has run out so
//	        it can't move." and send NOTHING (this is the Water of Life case)
//	    COutPacket(86 on v95) ; Encode4(get_update_time())
//	                          ; EncodeBuffer(&item->liCashItemSN, 8)
//	    SetExclRequestSent(1) ; YesNo "...purchase a new pet in the Cash Shop"
//	} else {
//	    COutPacket(110 on v95) -> the SPAWN_PET arm, see Spawn
//	}
//
// So this op means "the dried-up pet I just clicked has WZ noRevive; destroy
// the item." The pet is identified ONLY by its 8-byte cash serial — there is
// no slot on the wire.
//
// Body is version-stable across the range that has the op: v83 @0xa240a2 and
// v95 @0x9f6980 both encode tick + the raw 8-byte liCashItemSN, and the v92
// export records the same Encode4 + EncodeBuffer pair. v48/v61/v72/v79 have no
// opcode for it (registry: n-a).
//
// packet-audit:fname CWvsContext::SendActivatePetRequest
type DestroyItem struct {
	updateTime uint32
	// cashItemSerialNumber is GW_ItemSlotBase::liCashItemSN, encoded by
	// EncodeBuffer over the raw 8 bytes rather than by Encode8 — same wire
	// bytes, little-endian.
	cashItemSerialNumber uint64
}

func (m DestroyItem) UpdateTime() uint32 {
	return m.updateTime
}

func (m DestroyItem) CashItemSerialNumber() uint64 {
	return m.cashItemSerialNumber
}

func (m DestroyItem) Operation() string {
	return PetDestroyItemHandle
}

func (m DestroyItem) String() string {
	return fmt.Sprintf("updateTime [%d] cashItemSerialNumber [%d]", m.updateTime, m.cashItemSerialNumber)
}

func (m DestroyItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteLong(m.cashItemSerialNumber)
		return w.Bytes()
	}
}

func (m *DestroyItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.cashItemSerialNumber = r.ReadUint64()
	}
}
