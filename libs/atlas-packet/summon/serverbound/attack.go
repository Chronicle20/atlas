package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonAttackHandle = "SummonAttackHandle"

// AttackTarget is one damaged monster decoded from a summon ATTACK packet.
type AttackTarget struct {
	monsterOid uint32
	damage     uint32
	delay      int16
}

func (t AttackTarget) MonsterOid() uint32 { return t.monsterOid }
func (t AttackTarget) Damage() uint32     { return t.damage }
func (t AttackTarget) Delay() int16       { return t.delay }

// Attack is the client -> server summon ATTACK packet, decoded per Cosmic
// SummonDamageHandler.handlePacket (SummonDamageHandler.java:53-84):
//
//	int oid
//	skip(4)
//	byte direction
//	byte numAttacked
//	skip(8)                       // mob x,y and summon x,y
//	per target:
//	  int monsterOid
//	  skip(8)
//	  Point curPos  (short x, short y)
//	  Point nextPos (short x, short y)
//	  short delay
//	  int damage
//
// Cosmic only consumes oid, direction, monsterOid, delay and damage; the
// skipped position bytes are read/written as-is so the layout round-trips and
// real client packets are consumed without leftover bytes.
type Attack struct {
	oid       uint32
	direction byte
	targets   []AttackTarget
}

func (m Attack) Oid() uint32             { return m.oid }
func (m Attack) Direction() byte         { return m.direction }
func (m Attack) Targets() []AttackTarget { return m.targets }

func (m Attack) Operation() string { return SummonAttackHandle }

func (m Attack) String() string {
	return fmt.Sprintf("oid [%d], direction [%d], targets [%d]", m.oid, m.direction, len(m.targets))
}

func (m Attack) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		w.Skip(4)
		w.WriteByte(m.direction)
		w.WriteByte(byte(len(m.targets)))
		w.Skip(8) // mob x,y and summon x,y
		for _, t := range m.targets {
			w.WriteInt(t.monsterOid)
			w.Skip(8)
			w.Skip(4) // curPos (short x, short y)
			w.Skip(4) // nextPos (short x, short y)
			w.WriteInt16(t.delay)
			w.WriteInt(t.damage)
		}
		return w.Bytes()
	}
}

func (m *Attack) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		r.Skip(4)
		m.direction = r.ReadByte()
		count := int(r.ReadByte())
		r.Skip(8) // mob x,y and summon x,y
		m.targets = make([]AttackTarget, 0, count)
		for i := 0; i < count; i++ {
			monsterOid := r.ReadUint32()
			r.Skip(8)
			r.Skip(4) // curPos (short x, short y)
			r.Skip(4) // nextPos (short x, short y)
			delay := r.ReadInt16()
			damage := r.ReadUint32()
			m.targets = append(m.targets, AttackTarget{monsterOid: monsterOid, damage: damage, delay: delay})
		}
	}
}
