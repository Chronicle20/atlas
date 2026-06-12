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

// Damage is the client -> server summon DAMAGE packet, decoded per Cosmic
// DamageSummonHandler.handlePacket (DamageSummonHandler.java:35-38):
//
//	int oid
//	skip(1)              // -1, unused
//	int damage
//	int monsterIdFrom
type Damage struct {
	oid           uint32
	damage        uint32
	monsterIdFrom uint32
}

func NewDamage(oid, damage, monsterIdFrom uint32) Damage {
	return Damage{
		oid:           oid,
		damage:        damage,
		monsterIdFrom: monsterIdFrom,
	}
}

func (m Damage) Oid() uint32           { return m.oid }
func (m Damage) Damage() uint32        { return m.damage }
func (m Damage) MonsterIdFrom() uint32 { return m.monsterIdFrom }
func (m Damage) Operation() string     { return SummonDamageHandle }

func (m Damage) String() string {
	return fmt.Sprintf("oid [%d], damage [%d], monsterIdFrom [%d]", m.oid, m.damage, m.monsterIdFrom)
}

func (m Damage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		w.Skip(1) // -1, unused
		w.WriteInt(m.damage)
		w.WriteInt(m.monsterIdFrom)
		return w.Bytes()
	}
}

func (m *Damage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		r.Skip(1) // -1, unused
		m.damage = r.ReadUint32()
		m.monsterIdFrom = r.ReadUint32()
	}
}
