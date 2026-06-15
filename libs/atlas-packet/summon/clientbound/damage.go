package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SummonDamageWriter = "SummonDamage"

// SummonDamage is the server -> client summon DAMAGE packet. The wire layout is
// the IDB-confirmed CSummonedPool::OnSkill@0x7a6ebe reader (dispatched on the
// HIGHER of the swapped skill/damage opcodes — see summon-wire-truth.md):
//
//	int  cid              // summon owner character id (consumed by dispatcher)
//	int  oid              // v95+ only (gated >= 95); v83/v87 have NO oid
//	byte attackIdx        // fixed 12 (> -2, so the template branch always fires)
//	int  damage
//	if attackIdx > -2:
//	  int  monsterIdFrom  // attacking monster template id
//	  byte bLeft          // fixed 0
//
// The 12/0 constants mirror Cosmic; the only structural gate is the v95+ oid.
// The clientbound damage reader stops at bLeft on ALL versions — v83
// CSummonedPool::OnSkill@0x7a6ebe, v87 @0x7f969f, and v95 OnHit@0x74bc80 all
// read nothing after bLeft (the dir<0 byte belongs to the SERVERBOUND
// SetDamaged send, not this broadcast).
type SummonDamage struct {
	cid           uint32
	oid           uint32
	damage        uint32
	monsterIdFrom uint32
}

func NewSummonDamage(cid, oid, damage, monsterIdFrom uint32) SummonDamage {
	return SummonDamage{
		cid:           cid,
		oid:           oid,
		damage:        damage,
		monsterIdFrom: monsterIdFrom,
	}
}

func (m SummonDamage) Cid() uint32           { return m.cid }
func (m SummonDamage) Oid() uint32           { return m.oid }
func (m SummonDamage) Damage() uint32        { return m.damage }
func (m SummonDamage) MonsterIdFrom() uint32 { return m.monsterIdFrom }
func (m SummonDamage) Operation() string     { return SummonDamageWriter }

func (m SummonDamage) String() string {
	return fmt.Sprintf("cid [%d], oid [%d], damage [%d], monsterIdFrom [%d]", m.cid, m.oid, m.damage, m.monsterIdFrom)
}

func (m SummonDamage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		// oid: present on ALL versions. cid is read upstream by
		// CUserPool::OnUserCommonPacket; CSummonedPool::OnPacket@0x938dd7 then does
		// one Decode4 = the oid before the damage handler. Wire = cid + oid + body
		// (the old "no oid pre-95" reading missed the upstream cid read).
		w.WriteInt(m.oid)
		w.WriteByte(12) // attackIdx (> -2 so the template branch fires)
		w.WriteInt(m.damage)
		w.WriteInt(m.monsterIdFrom)
		w.WriteByte(0) // bLeft (final field — no trailing dir byte on any version)
		return w.Bytes()
	}
}

func (m *SummonDamage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		r.Skip(1) // attackIdx (12)
		m.damage = r.ReadUint32()
		m.monsterIdFrom = r.ReadUint32()
		r.Skip(1) // bLeft (final field — no trailing dir byte on any version)
	}
}
