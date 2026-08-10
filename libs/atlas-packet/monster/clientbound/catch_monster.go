package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CatchMonsterWriter = "CatchMonster"

// CatchMonster is the clientbound CATCH_MONSTER packet (CMob::OnCatchEffect):
// the server tells the client to play a mob-capture effect (the Pokemon-style
// "caught" animation on the targeted mob).
//
// Byte layout (IDA-verified — version-dependent):
//
//	v83/v84/v87/jms (single field):
//	  - result : byte — the catch-effect result code passed to ShowCatchEffect
//	v95 (two fields):
//	  - result  : byte — Decode1 -> ShowCatchEffect 1st arg
//	  - success : byte — Decode1; ShowCatchEffect 2nd arg = (success ? 0x10E : 0)
//
// IDA basis: CMob::OnCatchEffect —
//   - v83 @0x66d6b9: `v3 = Decode1(a1); ShowCatchEffect(this, v3)` — one Decode1.
//   - v84 @0x6839bb, v87 @0x6a8585: identical single-Decode1 shape.
//   - jms sub_6EAE5F @0x6eae5f (OnCatchEffect unnamed in jms IDB): one Decode1,
//     ShowCatchEffect's 2nd arg is uninitialised garbage (not read off the wire).
//   - v95 @0x63cd00: `v3 = Decode1; v4 = Decode1; ShowCatchEffect(this, v3,
//     v4 != 0 ? 0x10E : 0)` — two wire bytes. The extra success byte is a GMS-95
//     addition, so the branch gates on GMS region AND major >= 95.
//
// Wire note: CATCH_MONSTER is a per-mob OnMobPacket case, so the client consumes
// a leading uniqueId via CMobPool::OnMobPacket (Decode4 -> GetMob) BEFORE
// CMob::OnCatchEffect reads the result byte. This is universal, not legacy —
// confirmed on v48 @0x559390, v61 @0x5d48f3, v79 @0x646d46, v83 @0x67936d,
// v92 @0x64a6c0, v95 @0x6570b0, jms @0x6f8732, and by symbol on v84/v87
// (task-212 design.md §2 F-1). It was previously gated to pre-v83 by
// legacyMobPoolPrefix, which made every v83+ catch packet undecodable by the
// client; the gate is deleted.
//
// packet-audit:fname CMob::OnCatchEffect
type CatchMonster struct {
	uniqueId uint32
	result   byte
	success  byte
}

func NewCatchMonster(uniqueId uint32, result byte, success byte) CatchMonster {
	return CatchMonster{uniqueId: uniqueId, result: result, success: success}
}

func (m CatchMonster) UniqueId() uint32  { return m.uniqueId }
func (m CatchMonster) Result() byte      { return m.result }
func (m CatchMonster) Success() byte     { return m.success }
func (m CatchMonster) Operation() string { return CatchMonsterWriter }
func (m CatchMonster) String() string {
	return fmt.Sprintf("result [%d], success [%d]", m.result, m.success)
}

// v95CatchLayout reports whether this tenant uses the two-byte GMS-95 layout.
func v95CatchLayout(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(95)
}

func (m CatchMonster) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(m.result)
		if v95CatchLayout(t) {
			w.WriteByte(m.success)
		}
		return w.Bytes()
	}
}

func (m *CatchMonster) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.result = r.ReadByte()
		if v95CatchLayout(t) {
			m.success = r.ReadByte()
		}
	}
}
