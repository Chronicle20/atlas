package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonDamageWriter = "SummonDamage"

// SummonDamage is the server -> client summon DAMAGE packet, a faithful port of
// Cosmic PacketCreator.damageSummon (PacketCreator.java:4076):
//
//	int cid              // summon owner character id
//	int oid              // summon object id
//	byte 12              // fixed
//	int damage
//	int monsterIdFrom    // attacking monster id
//	byte 0               // fixed
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
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		w.WriteInt(m.oid)
		w.WriteByte(12)
		w.WriteInt(m.damage)
		w.WriteInt(m.monsterIdFrom)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *SummonDamage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.oid = r.ReadUint32()
		r.Skip(1) // byte 12
		m.damage = r.ReadUint32()
		m.monsterIdFrom = r.ReadUint32()
		r.Skip(1) // byte 0
	}
}
