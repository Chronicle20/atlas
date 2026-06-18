package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CatchMonsterWithItemWriter = "CatchMonsterWithItem"

// CatchMonsterWithItem is the clientbound CATCH_MONSTER_WITH_ITEM packet
// (CMob::OnEffectByItem): the server tells the client to play a capture-by-item
// effect (e.g. a catch item used on a mob) on the targeted mob.
//
// Byte layout (IDA-verified, identical across all 5 versions — Decode4 + Decode1):
//   - itemId : int32 — the catch item id (Decode4 -> ShowEffectByItem 1st arg)
//   - result : byte  — the effect result code (Decode1 -> ShowEffectByItem 2nd arg)
//
// IDA basis: CMob::OnEffectByItem — v83 @0x66d997 (`v3 = Decode4(a2); v4 =
// Decode1(a2); ShowEffectByItem(this, v3, v4)`), v84 @0x683c9f, v87 @0x6a886e,
// v95 @0x63cd40, jms @0x6eb148 — every version reads exactly one Decode4 then
// one Decode1.
//
// packet-audit:fname CMob::OnEffectByItem
type CatchMonsterWithItem struct {
	itemId int32
	result byte
}

func NewCatchMonsterWithItem(itemId int32, result byte) CatchMonsterWithItem {
	return CatchMonsterWithItem{itemId: itemId, result: result}
}

func (m CatchMonsterWithItem) ItemId() int32     { return m.itemId }
func (m CatchMonsterWithItem) Result() byte      { return m.result }
func (m CatchMonsterWithItem) Operation() string { return CatchMonsterWithItemWriter }
func (m CatchMonsterWithItem) String() string {
	return fmt.Sprintf("itemId [%d], result [%d]", m.itemId, m.result)
}

func (m CatchMonsterWithItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.itemId)
		w.WriteByte(m.result)
		return w.Bytes()
	}
}

func (m *CatchMonsterWithItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = r.ReadInt32()
		m.result = r.ReadByte()
	}
}
