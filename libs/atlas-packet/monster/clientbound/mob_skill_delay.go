package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobSkillDelayWriter = "MobSkillDelay"

// MobSkillDelay is the clientbound MOB_SKILL_DELAY packet (CMob::OnMobSkillDelay):
// the server schedules a delayed mob skill (m_delaySkill); the client starts the
// delay timer and remembers the skill to fire.
//
// Byte layout (IDA-verified, four Decode4):
//   - delay   : int32 — m_delaySkill.tSkillDelayTime = Decode4 (+ get_update_time)
//   - skillId : int32 — m_delaySkill.nSkillID = Decode4
//   - skillLevel : int32 — m_delaySkill.nSLV = Decode4
//   - option  : int32 — m_delaySkill.nOption = Decode4
//
// IDA basis: CMob::OnMobSkillDelay — v84 @0x688524 (dispatcher case 261),
// v87 @0x6ad0e8, v95 @0x63d560, jms @0x6ef0d4 (4× Decode4). VERSION-ABSENT in v83:
// the v83 dispatcher CMobPool::OnMobPacket @0x67936d ends at case 0xFF
// (OnMobAttackedByMob) with no skill-delay case — this is a later-version feature.
//
// packet-audit:fname CMob::OnMobSkillDelay
type MobSkillDelay struct {
	delay      int32
	skillId    int32
	skillLevel int32
	option     int32
}

func NewMobSkillDelay(delay int32, skillId int32, skillLevel int32, option int32) MobSkillDelay {
	return MobSkillDelay{delay: delay, skillId: skillId, skillLevel: skillLevel, option: option}
}

func (m MobSkillDelay) Delay() int32      { return m.delay }
func (m MobSkillDelay) SkillId() int32    { return m.skillId }
func (m MobSkillDelay) SkillLevel() int32 { return m.skillLevel }
func (m MobSkillDelay) Option() int32     { return m.option }
func (m MobSkillDelay) Operation() string { return MobSkillDelayWriter }
func (m MobSkillDelay) String() string {
	return fmt.Sprintf("delay [%d], skillId [%d], skillLevel [%d], option [%d]", m.delay, m.skillId, m.skillLevel, m.option)
}

func (m MobSkillDelay) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.delay)
		w.WriteInt32(m.skillId)
		w.WriteInt32(m.skillLevel)
		w.WriteInt32(m.option)
		return w.Bytes()
	}
}

func (m *MobSkillDelay) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.delay = r.ReadInt32()
		m.skillId = r.ReadInt32()
		m.skillLevel = r.ReadInt32()
		m.option = r.ReadInt32()
	}
}
