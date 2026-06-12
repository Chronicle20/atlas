package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonSpawnWriter = "SummonSpawn"

type SummonSpawn struct {
	ownerId      uint32
	oid          uint32
	skillId      uint32
	level        byte
	x            int16
	y            int16
	stance       byte
	movementType byte
	puppet       bool
	animated     bool
}

func NewSummonSpawn(ownerId, oid, skillId uint32, level byte, x, y int16, stance, movementType byte, puppet, animated bool) SummonSpawn {
	return SummonSpawn{
		ownerId:      ownerId,
		oid:          oid,
		skillId:      skillId,
		level:        level,
		x:            x,
		y:            y,
		stance:       stance,
		movementType: movementType,
		puppet:       puppet,
		animated:     animated,
	}
}

func (m SummonSpawn) OwnerId() uint32    { return m.ownerId }
func (m SummonSpawn) Oid() uint32        { return m.oid }
func (m SummonSpawn) SkillId() uint32    { return m.skillId }
func (m SummonSpawn) Level() byte        { return m.level }
func (m SummonSpawn) X() int16           { return m.x }
func (m SummonSpawn) Y() int16           { return m.y }
func (m SummonSpawn) Stance() byte       { return m.stance }
func (m SummonSpawn) MovementType() byte { return m.movementType }
func (m SummonSpawn) Puppet() bool       { return m.puppet }
func (m SummonSpawn) Animated() bool     { return m.animated }
func (m SummonSpawn) Operation() string  { return SummonSpawnWriter }
func (m SummonSpawn) String() string {
	return fmt.Sprintf("ownerId [%d], oid [%d], skillId [%d], level [%d]", m.ownerId, m.oid, m.skillId, m.level)
}

func (m SummonSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt(m.oid)
		w.WriteInt(m.skillId)
		w.WriteByte(0x0A) // v83 marker; per-version value confirmed via IDA in Phase 6
		w.WriteByte(m.level)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0)
		w.WriteByte(m.movementType)
		w.WriteBool(!m.puppet)   // attack flag = !isPuppet
		w.WriteBool(!m.animated) // !animated
		return w.Bytes()
	}
}

func (m *SummonSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.oid = r.ReadUint32()
		m.skillId = r.ReadUint32()
		_ = r.ReadByte() // 0x0A v83 marker; per-version value confirmed via IDA in Phase 6
		m.level = r.ReadByte()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.stance = r.ReadByte()
		_ = r.ReadUint16() // reserved short 0
		m.movementType = r.ReadByte()
		m.puppet = !r.ReadBool()   // attack flag = !isPuppet
		m.animated = !r.ReadBool() // !animated
	}
}
