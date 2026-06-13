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
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		w.WriteByte(0) // attackIdx (-1/unused in Cosmic baseline; 0-fill, was Skip(1))
		w.WriteInt(m.damage)
		w.WriteInt(m.monsterIdFrom)
		// v95+ DELTA (gated >= 95, GMS only): the v95 client send site
		// CSummoned::SetDamaged@0x74b730 emits a trailing Encode1(nDir<0) after
		// the mob-template id (Encode1@0x74bbed). v87's SetDamaged@0x7f879a is
		// byte-identical (also has the trailing dir byte), but the Cosmic v83
		// baseline reads only oid+skip1+damage+monsterIdFrom, so the trailing
		// byte is gated >=95 to avoid touching the v83/v87 decode path. See
		// summon-packet-delta.md §3.5.
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			w.WriteByte(0) // dir<0 flag (0-fill)
		}
		return w.Bytes()
	}
}

func (m *Damage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		_ = r.ReadByte() // attackIdx (-1/unused in Cosmic baseline; was Skip(1))
		m.damage = r.ReadUint32()
		m.monsterIdFrom = r.ReadUint32()
		// v95+ DELTA (mirror of Encode): consume the trailing dir<0 flag byte.
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			_ = r.ReadByte() // dir<0 flag
		}
	}
}
