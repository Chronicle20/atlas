package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		// oid: the summon object id, present on ALL versions (cid, oid, skillId) —
		// matches Cosmic spawnSummon (ownerId, objectId, skillId). The ACTIVE client
		// dispatch (v83 field path → OnCreated @0x95ADEC) has the DISPATCHER consume
		// the leading cid, so OnCreated then reads oid, skillId, charLevel, SLV. This
		// was confirmed live in x32dbg: at OnCreated's first Decode4 the read offset
		// is already past cid (0x0A), so the int after cid MUST be the oid — omitting
		// it makes the client read skillId into the cid slot and starve at the
		// foothold Decode2 (client closes). The earlier "no oid pre-95" reading
		// analyzed the INACTIVE OnCreated @0x938F61, whose dispatcher does NOT
		// pre-read cid — the wrong path. See summon-wire-truth.md spawn row.
		w.WriteInt(m.oid)
		w.WriteInt(m.skillId)
		// v83 "0x0A marker" is semantically the charLevel byte; the following
		// "reserved short 0" is the foothold id. Both are visual-only and the
		// fixed writes are client-tolerated. See summon-packet-delta.md §3.1
		// (CSummoned::Init@0x755740, IDA-confirmed).
		w.WriteByte(0x0A) // charLevel (visual-only)
		// SLV byte: present on GMS v83+ and JMS v185, ABSENT on GMS v79. The v79
		// spawn reader CSummonedPool::OnCreated (sub_89268A@0x89268a) reads only
		// Decode4(oid), Decode4(skillId), Decode1(charLevel) before descending into
		// the Init blob (sub_719F7B@0x719f7b, first read Decode2(x)) — i.e. ONE byte
		// between skillId and x, where v83+ read TWO (charLevel + SLV). Writing the
		// extra SLV byte on v79 misaligns the client's Decode2(x). See spawnHasSkillLevel.
		if spawnHasSkillLevel(t) {
			w.WriteByte(m.level)
		}
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0) // foothold id (visual-only)
		w.WriteByte(m.movementType)
		w.WriteBool(!m.puppet)   // attack flag = !isPuppet
		w.WriteBool(!m.animated) // !animated
		// avatar-look DELTA: the spawn Init blob ends with a byte bAvatarLook,
		// then an AvatarLook blob iff present, then a Tesla-Coil-only triangle
		// tail. Present on GMS v95+ (CSummoned::Init@0x755740) AND on JMS v185
		// (jms185 spawn Init reader sub_823AED@0x823aed: Decode1 bAvatarLook
		// @0x823b99, then `if (v8) AvatarLook::Decode` @0x823bb0 — IDB-confirmed).
		// It is ABSENT on GMS v83/v84/v87 (only ONE int + the fixed Init tail; no
		// avatar byte). None of the 21 v83-roster summons carry an avatar look and
		// Tesla Coil is out of roster, so we write present = 0 and the client skips
		// both the blob and the triangle tail. See spawnHasAvatarLook.
		if spawnHasAvatarLook(t) {
			w.WriteByte(0) // bAvatarLook present = 0 (no AvatarLook blob, no Tesla tail)
		}
		return w.Bytes()
	}
}

func (m *SummonSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		m.skillId = r.ReadUint32()
		_ = r.ReadByte() // charLevel (visual-only); see summon-packet-delta.md §3.1
		if spawnHasSkillLevel(t) {
			m.level = r.ReadByte() // SLV — absent on GMS v79 (see Encode / spawnHasSkillLevel)
		}
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.stance = r.ReadByte()
		_ = r.ReadUint16() // foothold id (visual-only)
		m.movementType = r.ReadByte()
		m.puppet = !r.ReadBool()   // attack flag = !isPuppet
		m.animated = !r.ReadBool() // !animated
		// avatar-look DELTA (mirror of Encode): read the bAvatarLook present byte
		// on GMS v95+ and JMS v185. For our roster it is always 0, so no
		// AvatarLook blob / Tesla tail follows. See spawnHasAvatarLook.
		if spawnHasAvatarLook(t) {
			_ = r.ReadByte() // bAvatarLook present (0 for our roster)
		}
	}
}

// spawnHasAvatarLook reports whether the spawn Init blob carries the trailing
// bAvatarLook present-byte (+ optional AvatarLook blob + Tesla triangle tail).
// Present on GMS v95+ (CSummoned::Init@0x755740) and on JMS v185 (spawn Init
// reader sub_823AED@0x823aed Decode1 bAvatarLook@0x823b99). Absent on GMS
// v83/v84/v87 (no avatar byte in the Init tail — IDB-confirmed).
func spawnHasAvatarLook(t tenant.Model) bool {
	if t.IsRegion("JMS") {
		return t.MajorAtLeast(185)
	}
	return t.IsRegion("GMS") && t.MajorAtLeast(95)
}

// spawnHasSkillLevel reports whether the spawn Init prefix carries the SLV byte
// after charLevel. Present on GMS v83+ and JMS v185 (the verified versions all
// read charLevel + SLV, two bytes, before the x/y Init blob); ABSENT on GMS v79,
// whose spawn reader CSummonedPool::OnCreated (sub_89268A@0x89268a) reads only a
// single charLevel byte (Decode1@0x8926b9) before the blob reader sub_719F7B@0x719f7b
// begins at Decode2(x). Boundary is verified at v79 (absent) / v83 (present);
// MajorAtLeast(83) places the gate at the first version known to carry it.
func spawnHasSkillLevel(t tenant.Model) bool {
	return t.MajorAtLeast(83)
}
