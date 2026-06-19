package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillPrepareForeignWriter = "CharacterSkillPrepareForeign"

// SkillPrepareForeign encodes/decodes the clientbound remote skill-prepare
// packet (OnSkillPrepare). Wire-spec §3: the dispatcher reads charId u32 first;
// the relay packet must therefore lead with charId before the handler body fields.
//
// Full wire order: charId u32, skillId u32, level u8, action u16, actionSpeed u8.
// Field order and widths are identical across all five versions (v83/v84/v87/v95/jms185).
// packet-audit:fname CUserRemote::OnSkillPrepare
type SkillPrepareForeign struct {
	characterId uint32
	skillId     uint32
	level       byte
	action      uint16
	actionSpeed byte
}

func NewSkillPrepareForeign(characterId uint32, skillId uint32, level byte, action uint16, actionSpeed byte) SkillPrepareForeign {
	return SkillPrepareForeign{
		characterId: characterId,
		skillId:     skillId,
		level:       level,
		action:      action,
		actionSpeed: actionSpeed,
	}
}

func (m SkillPrepareForeign) CharacterId() uint32  { return m.characterId }
func (m SkillPrepareForeign) SkillId() uint32     { return m.skillId }
func (m SkillPrepareForeign) Level() byte         { return m.level }
func (m SkillPrepareForeign) Action() uint16      { return m.action }
func (m SkillPrepareForeign) ActionSpeed() byte   { return m.actionSpeed }
func (m SkillPrepareForeign) Operation() string   { return CharacterSkillPrepareForeignWriter }

func (m SkillPrepareForeign) String() string {
	return fmt.Sprintf("foreign skill prepare characterId [%d] skillId [%d] level [%d] action [%d] actionSpeed [%d]",
		m.characterId, m.skillId, m.level, m.action, m.actionSpeed)
}

func (m SkillPrepareForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *SkillPrepareForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.level = r.ReadByte()
		m.action = r.ReadUint16()
		m.actionSpeed = r.ReadByte()
	}
}
