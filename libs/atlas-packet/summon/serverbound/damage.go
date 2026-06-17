package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonDamageHandle = "SummonDamageHandle"

// Damage is the client -> server summon DAMAGE packet (a puppet reporting that a
// mob hit it), decoded from the real client SEND site CSummoned::SetDamaged.
// The v83 (@0x7a607a), v87 (@0x7f879a) and v95 (@0x74b730) sends are
// byte-for-byte identical in body shape — the only per-version difference is the
// summon identity semantics (cid vs summon id) and the opcode (routed by the
// socket layer, not consumed here).
//
//	Encode4 summonId                       ; v83/v87 = owner cid [obj+0xAC]; v95 = m_dwSummonedID
//	if (mob present):
//	  Encode1 attackIdx                    ; mob attack index
//	  Encode4 damage
//	  Encode4 monsterIdFrom                ; mob TEMPLATE id (fused dwTemplateID), not an oid
//	  Encode1 (dir < 0)                    ; impact direction flag — present in v83 too
//	else:
//	  Encode1 0xFE                         ; sentinel "-2" (no source mob)
//	  Encode4 damage
//
// The trailing dir byte and the 0xFE no-mob branch were both missing from the
// prior Cosmic-derived decoder; the ASM (v83 Encode1@0x7a62f4 / 0x7a62a8) proves
// both exist on v83. The dir byte is consumed but not surfaced (the server does
// not need it).
// packet-audit:fname CSummonedPool::OnHit
type Damage struct {
	summonId      uint32
	attackIdx     byte
	damage        uint32
	monsterIdFrom uint32
}

func NewDamage(summonId, damage, monsterIdFrom uint32) Damage {
	return Damage{
		summonId:      summonId,
		attackIdx:     0,
		damage:        damage,
		monsterIdFrom: monsterIdFrom,
	}
}

func (m Damage) SummonId() uint32      { return m.summonId }
func (m Damage) AttackIdx() byte       { return m.attackIdx }
func (m Damage) Damage() uint32        { return m.damage }
func (m Damage) MonsterIdFrom() uint32 { return m.monsterIdFrom }
func (m Damage) Operation() string     { return SummonDamageHandle }

func (m Damage) String() string {
	return fmt.Sprintf("summonId [%d], attackIdx [%d], damage [%d], monsterIdFrom [%d]", m.summonId, m.attackIdx, m.damage, m.monsterIdFrom)
}

func (m Damage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.summonId)
		if m.attackIdx == 0xFE {
			w.WriteByte(0xFE)
			w.WriteInt(m.damage)
			return w.Bytes()
		}
		w.WriteByte(m.attackIdx)
		w.WriteInt(m.damage)
		w.WriteInt(m.monsterIdFrom)
		w.WriteByte(0) // dir<0 flag (0-fill)
		return w.Bytes()
	}
}

func (m *Damage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.summonId = r.ReadUint32()
		m.attackIdx = r.ReadByte()
		if m.attackIdx == 0xFE {
			// No source mob: only the damage follows. monsterIdFrom stays 0.
			m.damage = r.ReadUint32()
			return
		}
		m.damage = r.ReadUint32()
		m.monsterIdFrom = r.ReadUint32()
		_ = r.ReadByte() // dir<0 flag (v83 Encode1@0x7a62f4) — consumed, not surfaced
	}
}
