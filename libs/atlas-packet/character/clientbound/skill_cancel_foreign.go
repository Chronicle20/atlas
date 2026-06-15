package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillCancelForeignWriter = "CharacterSkillCancelForeign"

// CharacterSkillCancelForeign encodes/decodes the clientbound remote skill-cancel
// packet (OnSkillCancel). Wire-spec §4: the dispatcher reads charId u32 first;
// the relay packet must therefore lead with charId before the handler body fields.
//
// Full wire order: charId u32, skillId u32.
// Field order and widths are identical across all five versions (v83/v84/v87/v95/jms185).
type CharacterSkillCancelForeign struct {
	characterId uint32
	skillId     uint32
}

func NewCharacterSkillCancelForeign(characterId uint32, skillId uint32) CharacterSkillCancelForeign {
	return CharacterSkillCancelForeign{
		characterId: characterId,
		skillId:     skillId,
	}
}

func (m CharacterSkillCancelForeign) CharacterId() uint32 { return m.characterId }
func (m CharacterSkillCancelForeign) SkillId() uint32    { return m.skillId }
func (m CharacterSkillCancelForeign) Operation() string  { return CharacterSkillCancelForeignWriter }

func (m CharacterSkillCancelForeign) String() string {
	return fmt.Sprintf("foreign skill cancel characterId [%d] skillId [%d]", m.characterId, m.skillId)
}

func (m CharacterSkillCancelForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		return w.Bytes()
	}
}

func (m *CharacterSkillCancelForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.skillId = r.ReadUint32()
	}
}
