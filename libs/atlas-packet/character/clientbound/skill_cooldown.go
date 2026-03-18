package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillCooldownWriter = "CharacterSkillCooldown"

type CharacterSkillCooldown struct {
	skillId  uint32
	cooldown uint16
}

func NewCharacterSkillCooldown(skillId uint32, cooldown uint16) CharacterSkillCooldown {
	return CharacterSkillCooldown{skillId: skillId, cooldown: cooldown}
}

func (m CharacterSkillCooldown) SkillId() uint32   { return m.skillId }
func (m CharacterSkillCooldown) Cooldown() uint16  { return m.cooldown }
func (m CharacterSkillCooldown) Operation() string  { return CharacterSkillCooldownWriter }
func (m CharacterSkillCooldown) String() string {
	return fmt.Sprintf("skillId [%d], cooldown [%d]", m.skillId, m.cooldown)
}

func (m CharacterSkillCooldown) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.skillId)
		w.WriteShort(m.cooldown)
		return w.Bytes()
	}
}

func (m *CharacterSkillCooldown) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadUint32()
		m.cooldown = r.ReadUint16()
	}
}
