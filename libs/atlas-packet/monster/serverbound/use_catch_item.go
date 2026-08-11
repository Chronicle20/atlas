package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const UseCatchItemHandle = "MonsterCatchItemUseHandle"

// UseCatchItem is the serverbound USE_CATCH_ITEM packet
// (CWvsContext::SendBridleItemUseRequest): the client asks the server to use a
// bridle/catch consumable (item class 227) on a specific field monster.
//
// Byte layout (IDA-verified, identical on every version inspected):
//   - updateTime      : uint32 — client get_update_time()
//   - slot            : int16  — inventory position (nPOS)
//   - itemId          : uint32 — the catch item (nItemID, always /10000 == 227)
//   - monsterUniqueId : uint32 — the hit mob's field object id
//
// IDA basis: CWvsContext::SendBridleItemUseRequest — gms_v48 @0x70e0c5,
// gms_v61 @0x832005, gms_v72 @0x90457d, gms_v79 @0x9558e5, gms_v83 @0xa09bdf,
// gms_v84 @0xa53fc1 (renamed live from sub_A53FC1; task-212), gms_v87
// @0xa9f48b, gms_v92 @0x9b5830, gms_v95 @0x9e08c0, jms_v185 @0xaee887. No
// version-gated divergence was observed, so this codec carries NO
// MajorAtLeast gate; introduce one only if a remaining IDB proves otherwise.
//
// The client sets ExclRequest immediately after the COutPacket ctor on every
// version inspected, so EVERY terminal server outcome must send an unlocking
// packet (design.md §4.6).
//
// packet-audit:fname CWvsContext::SendBridleItemUseRequest
type UseCatchItem struct {
	updateTime      uint32
	slot            int16
	itemId          uint32
	monsterUniqueId uint32
}

func NewUseCatchItem(updateTime uint32, slot int16, itemId uint32, monsterUniqueId uint32) UseCatchItem {
	return UseCatchItem{updateTime: updateTime, slot: slot, itemId: itemId, monsterUniqueId: monsterUniqueId}
}

func (m UseCatchItem) UpdateTime() uint32      { return m.updateTime }
func (m UseCatchItem) Slot() int16             { return m.slot }
func (m UseCatchItem) ItemId() uint32          { return m.itemId }
func (m UseCatchItem) MonsterUniqueId() uint32 { return m.monsterUniqueId }
func (m UseCatchItem) Operation() string       { return UseCatchItemHandle }
func (m UseCatchItem) String() string {
	return fmt.Sprintf("updateTime [%d], slot [%d], itemId [%d], monsterUniqueId [%d]", m.updateTime, m.slot, m.itemId, m.monsterUniqueId)
}

func (m UseCatchItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		w.WriteInt(m.monsterUniqueId)
		return w.Bytes()
	}
}

func (m *UseCatchItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.monsterUniqueId = r.ReadUint32()
	}
}
