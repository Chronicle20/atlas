package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
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
// Legacy (pre-v83) wire note: CATCH_MONSTER is a per-mob OnMobPacket case (op
// 229), so the v79 client consumes a leading uniqueId via CMobPool::OnMobPacket
// @0x646d46 (Decode4 @0x646d50 -> GetMob) BEFORE CMob::OnCatchEffect reads the
// result byte. See legacyMobPoolPrefix.
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

// legacyMobPoolPrefix reports whether the tenant prepends the per-mob uniqueId
// that CMobPool::OnMobPacket consumes (Decode4 -> GetMob) before dispatching a
// per-mob clientbound packet (CATCH_MONSTER / MONSTER_SPECIAL_EFFECT_BY_SKILL /
// INC_MOB_CHARGE_COUNT) to its CMob handler. VERIFIED for v79: CMobPool::OnMobPacket
// @0x646d46, Decode4 @0x646d50.
//
// Gated to the pre-v83 legacy range per the task-113 campaign freeze on v83+.
// NOTE: the sibling per-mob packets MonsterHealth (op 228) and MonsterMovement
// (op 217) — same dispatcher — already carry this uniqueId prefix UNCONDITIONALLY
// for all versions, so v83+ very likely need it here too; they are left frozen
// (unchanged) per campaign scope and flagged for a follow-up that can confirm it
// against the v83/v84/v87/v95/jms IDBs.
func legacyMobPoolPrefix(t tenant.Model) bool {
	return t.IsRegion("GMS") && !t.MajorAtLeast(83)
}

func (m CatchMonster) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if legacyMobPoolPrefix(t) {
			w.WriteInt(m.uniqueId)
		}
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
		if legacyMobPoolPrefix(t) {
			m.uniqueId = r.ReadUint32()
		}
		m.result = r.ReadByte()
		if v95CatchLayout(t) {
			m.success = r.ReadByte()
		}
	}
}
