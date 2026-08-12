package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CatchMonsterWithItemWriter = "CatchMonsterWithItem"

// CatchMonsterWithItem is the clientbound CATCH_MONSTER_WITH_ITEM packet
// (CMob::OnEffectByItem): the server tells the client to play a capture-by-item
// effect on the targeted mob.
//
// Byte layout (IDA-verified):
//   - uniqueId : int32 — the mob object id, consumed by CMobPool::OnMobPacket
//     (Decode4 -> GetMob) BEFORE dispatch. Universal, every version.
//   - itemId   : int32 — the catch item id (Decode4 -> ShowEffectByItem 1st arg)
//   - result   : byte  — the effect result code (Decode1 -> 2nd arg).
//     ABSENT on gms_v48 (see v48CatchByItemNoResult).
//
// IDA basis: CMobPool::OnMobPacket — v48 @0x559390, v61 @0x5d48f3, v79 @0x646d46,
// v83 @0x67936d, v92 @0x64a6c0, v95 @0x6570b0, jms @0x6f8732 (task-212 §2 F-1).
// CMob::OnEffectByItem — v61 @0x5cc793, v79 @0x63c937, v83 @0x66d997,
// v84 @0x683c9f, v87 @0x6a886e, v92 @0x630c50, v95 @0x63cd40, jms @0x6eb148;
// v48's arm is sub_551481, which reads Decode4 and nothing else (§2 F-2).
//
// packet-audit:fname CMob::OnEffectByItem
type CatchMonsterWithItem struct {
	uniqueId uint32
	itemId   int32
	result   byte
}

func NewCatchMonsterWithItem(uniqueId uint32, itemId int32, result byte) CatchMonsterWithItem {
	return CatchMonsterWithItem{uniqueId: uniqueId, itemId: itemId, result: result}
}

func (m CatchMonsterWithItem) UniqueId() uint32  { return m.uniqueId }
func (m CatchMonsterWithItem) ItemId() int32     { return m.itemId }
func (m CatchMonsterWithItem) Result() byte      { return m.result }
func (m CatchMonsterWithItem) Operation() string { return CatchMonsterWithItemWriter }
func (m CatchMonsterWithItem) String() string {
	return fmt.Sprintf("uniqueId [%d], itemId [%d], result [%d]", m.uniqueId, m.itemId, m.result)
}

// v48CatchByItemNoResult reports whether the tenant omits OnEffectByItem's
// trailing result byte. VERIFIED: v48's arm sub_551481 @0x551481 reads Decode4
// and nothing else; every later version reads Decode4 + Decode1 (v61 @0x5cc793,
// v79 @0x63c937, v92 @0x630c50, and v83/v84/v87/v95/jms per the addresses above).
func v48CatchByItemNoResult(t tenant.Model) bool {
	return t.IsRegion("GMS") && !t.MajorAtLeast(61)
}

func (m CatchMonsterWithItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteInt32(m.itemId)
		if !v48CatchByItemNoResult(t) {
			w.WriteByte(m.result)
		}
		return w.Bytes()
	}
}

func (m *CatchMonsterWithItem) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.itemId = r.ReadInt32()
		if !v48CatchByItemNoResult(t) {
			m.result = r.ReadByte()
		}
	}
}
