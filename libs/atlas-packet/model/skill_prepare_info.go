package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// swallowMobPrepareSkillId is the skill whose serverbound prepare packet
// carries a trailing swallowMobId on GMS v95+ / JMS (IDA guard
// `nSkillID == 33101005` in DoActiveSkill_Prepare). Job attribution is not
// verified, so the name describes the wire behavior, not the skill's identity.
const swallowMobPrepareSkillId uint32 = 33101005

// hasSwallowField reports whether the given tenant version includes the trailing
// conditional swallowMobId u32 in the serverbound prepare packet.
// Wire-spec §1: present ONLY on GMS v95+ and JMS (all versions); absent on v83/v84/v87.
func hasSwallowField(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 95) || t.Region() == "JMS"
}

// skillPrepareActionIsByte reports whether the action/direction field rides the wire
// as a single byte (bit7 = bLeft, bits0-6 = nAction) instead of a 2-byte short.
// IDA-verified: v72 CUserLocal::DoActiveSkill_Prepare @0x875535 Encode1s
// `(bLeft<<7)|(nAction&0x7F)` — one byte; GMS v79+ (v79 @0x8c17f2 fixture, action
// 0x0142) and JMS write a 2-byte short. Mirrors the CUserLocal attack action-width
// transition (the same nAction field is shared by attack + skill-prepare senders).
func skillPrepareActionIsByte(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 79
}

// ─── SkillPrepareInfo ─────────────────────────────────────────────────────────

// SkillPrepareInfo holds the body of a serverbound DoActiveSkill_Prepare packet.
// Wire order (all versions): skillId u32, level u8, action u16, actionSpeed u8.
// On GMS v95+ and JMS: if skillId == 33101005, a trailing swallowMobId u32 follows.
type SkillPrepareInfo struct {
	skillId      uint32
	level        byte
	action       uint16
	actionSpeed  byte
	swallowMobId uint32
}

func NewSkillPrepareInfo() *SkillPrepareInfo {
	return &SkillPrepareInfo{}
}

// Getters

func (m *SkillPrepareInfo) SkillId() uint32      { return m.skillId }
func (m *SkillPrepareInfo) Level() byte          { return m.level }
func (m *SkillPrepareInfo) Action() uint16       { return m.action }
func (m *SkillPrepareInfo) ActionSpeed() byte    { return m.actionSpeed }
func (m *SkillPrepareInfo) SwallowMobId() uint32 { return m.swallowMobId }

// Setters (builder-style, return *SkillPrepareInfo for chaining)

func (m *SkillPrepareInfo) SetSkillId(skillId uint32) *SkillPrepareInfo {
	m.skillId = skillId
	return m
}

func (m *SkillPrepareInfo) SetLevel(level byte) *SkillPrepareInfo {
	m.level = level
	return m
}

func (m *SkillPrepareInfo) SetAction(action uint16) *SkillPrepareInfo {
	m.action = action
	return m
}

func (m *SkillPrepareInfo) SetActionSpeed(actionSpeed byte) *SkillPrepareInfo {
	m.actionSpeed = actionSpeed
	return m
}

func (m *SkillPrepareInfo) SetSwallowMobId(swallowMobId uint32) *SkillPrepareInfo {
	m.swallowMobId = swallowMobId
	return m
}

// Encode serializes the serverbound prepare body.
// Wire-spec §1 field order: skillId u32, level u8, action u16, actionSpeed u8
// [, swallowMobId u32 when (GMS v95+ || JMS) && skillId == 33101005].
func (m *SkillPrepareInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(m.skillId)
		w.WriteByte(m.level)
		if skillPrepareActionIsByte(t) {
			w.WriteByte(byte(m.action & 0xFF)) // legacy pre-79 GMS: 1-byte action
		} else {
			w.WriteShort(m.action)
		}
		w.WriteByte(m.actionSpeed)
		if hasSwallowField(t) && m.skillId == swallowMobPrepareSkillId {
			w.WriteInt(m.swallowMobId)
		}
		return w.Bytes()
	}
}

// Decode deserializes the serverbound prepare body from r.
// Mirror of Encode — must consume the exact same bytes.
func (m *SkillPrepareInfo) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadUint32()
		m.level = r.ReadByte()
		if skillPrepareActionIsByte(t) {
			m.action = uint16(r.ReadByte())
		} else {
			m.action = r.ReadUint16()
		}
		m.actionSpeed = r.ReadByte()
		if hasSwallowField(t) && m.skillId == swallowMobPrepareSkillId {
			m.swallowMobId = r.ReadUint32()
		}
	}
}
