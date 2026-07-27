package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobSkillDelayEndHandle = "MobSkillDelayEnd"

// MobSkillDelayEnd is the serverbound MOB_SKILL_DELAY_END packet, built by
// CMob::Update when a mob's queued skill cast-delay elapses; the controller tells
// the server the mob is ready to fire so the skill effect can be applied.
//
// Byte layout (IDA-verified — the skill-delay-end COutPacket build site inside
// CMob::Update; four Encode4):
//   - mobCrc     : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//   - skillId    : uint32 — the queued mob skill id
//   - skillLevel : uint32 — the queued mob skill level
//   - value      : uint32 — the skill's option/charge value
//
// IDA basis: CMob::Update skill-delay-end send site — v87 @0x6a1c8d (opcode 0xCF),
// v95 @0x6543d1 (opcode 0xEA), jms @0x6e3de5:
//
//	COutPacket(op); Encode4(SecureFuse(m_dwMobID)); Encode4(m_nSkillID);
//	Encode4(m_nSkillLevel); Encode4(m_nSkillOption); SendPacket
//
// v84 emits this at opcode 0xC8 @0x67d534 (registry was csv-import-stale at 0xC3).
//
// VERSION delta: v83 has NO sender for this op. The v83 CMob::Update builds no
// COutPacket at this opcode (the skill-delay-end feature post-dates v83, mirroring
// its clientbound twin MOB_SKILL_DELAY which the v83 dispatcher also lacks — see
// structures/applicability.md). v84/v87/v95/jms all carry the four-Encode4 send.
//
// packet-audit:fname CMob::Update
type MobSkillDelayEnd struct {
	mobCrc     uint32
	skillId    uint32
	skillLevel uint32
	value      uint32
}

func (m MobSkillDelayEnd) MobCrc() uint32     { return m.mobCrc }
func (m MobSkillDelayEnd) SkillId() uint32    { return m.skillId }
func (m MobSkillDelayEnd) SkillLevel() uint32 { return m.skillLevel }
func (m MobSkillDelayEnd) Value() uint32      { return m.value }
func (m MobSkillDelayEnd) Operation() string  { return MobSkillDelayEndHandle }
func (m MobSkillDelayEnd) String() string {
	return fmt.Sprintf("mobCrc [%d], skillId [%d], skillLevel [%d], value [%d]", m.mobCrc, m.skillId, m.skillLevel, m.value)
}

func (m MobSkillDelayEnd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		w.WriteInt(m.skillId)
		w.WriteInt(m.skillLevel)
		w.WriteInt(m.value)
		return w.Bytes()
	}
}

func (m *MobSkillDelayEnd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.skillLevel = r.ReadUint32()
		m.value = r.ReadUint32()
	}
}
