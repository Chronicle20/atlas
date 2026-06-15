package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillPrepareForeignWriter = "CharacterSkillPrepareForeign"

// CharacterSkillPrepareForeign encodes/decodes the clientbound remote skill-prepare
// packet (OnSkillPrepare). Wire-spec §3: the dispatcher reads charId u32 first;
// the relay packet must therefore lead with charId before the handler body fields.
//
// Full wire order: charId u32, skillId u32, level u8, action u16, actionSpeed u8.
// Field order and widths are identical across all five versions (v83/v84/v87/v95/jms185).
type CharacterSkillPrepareForeign struct {
	characterId uint32
	skillId     uint32
	level       byte
	action      uint16
	actionSpeed byte
}

func NewCharacterSkillPrepareForeign(characterId uint32, skillId uint32, level byte, action uint16, actionSpeed byte) CharacterSkillPrepareForeign {
	return CharacterSkillPrepareForeign{
		characterId: characterId,
		skillId:     skillId,
		level:       level,
		action:      action,
		actionSpeed: actionSpeed,
	}
}

func (m CharacterSkillPrepareForeign) CharacterId() uint32  { return m.characterId }
func (m CharacterSkillPrepareForeign) SkillId() uint32     { return m.skillId }
func (m CharacterSkillPrepareForeign) Level() byte         { return m.level }
func (m CharacterSkillPrepareForeign) Action() uint16      { return m.action }
func (m CharacterSkillPrepareForeign) ActionSpeed() byte   { return m.actionSpeed }
func (m CharacterSkillPrepareForeign) Operation() string   { return CharacterSkillPrepareForeignWriter }

func (m CharacterSkillPrepareForeign) String() string {
	return fmt.Sprintf("foreign skill prepare characterId [%d] skillId [%d] level [%d] action [%d] actionSpeed [%d]",
		m.characterId, m.skillId, m.level, m.action, m.actionSpeed)
}

func (m CharacterSkillPrepareForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		w.WriteByte(m.level)
		w.WriteShort(m.action)
		w.WriteByte(m.actionSpeed)
		return w.Bytes()
	}
}

func (m *CharacterSkillPrepareForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.level = r.ReadByte()
		m.action = r.ReadUint16()
		m.actionSpeed = r.ReadByte()
	}
}
