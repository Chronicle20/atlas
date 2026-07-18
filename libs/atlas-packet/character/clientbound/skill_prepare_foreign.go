package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// skillPrepareForeignActionIsByte reports whether the action/direction field rides
// the wire as a single byte (bit7 = bLeft, bits0-6 = nAction) instead of a 2-byte
// short. IDA-verified: v72 CUserRemote::OnSkillPrepare @0x889ec7 reads Decode1 then
// >>7 / &0x7F — one byte; GMS v79+ (v79 @0x8d6cd6 fixture, action 0x0142) and JMS
// read a 2-byte short. Mirrors the CUserRemote::OnAttack action-width transition.
func skillPrepareForeignActionIsByte(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.Region() == "GMS" && t.MajorVersion() < 79
}

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

func (m SkillPrepareForeign) CharacterId() uint32 { return m.characterId }
func (m SkillPrepareForeign) SkillId() uint32     { return m.skillId }
func (m SkillPrepareForeign) Level() byte         { return m.level }
func (m SkillPrepareForeign) Action() uint16      { return m.action }
func (m SkillPrepareForeign) ActionSpeed() byte   { return m.actionSpeed }
func (m SkillPrepareForeign) Operation() string   { return CharacterSkillPrepareForeignWriter }

func (m SkillPrepareForeign) String() string {
	return fmt.Sprintf("foreign skill prepare characterId [%d] skillId [%d] level [%d] action [%d] actionSpeed [%d]",
		m.characterId, m.skillId, m.level, m.action, m.actionSpeed)
}

func (m SkillPrepareForeign) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		w.WriteByte(m.level)
		if skillPrepareForeignActionIsByte(ctx) {
			w.WriteByte(byte(m.action & 0xFF)) // legacy pre-79 GMS: 1-byte action
		} else {
			w.WriteShort(m.action)
		}
		w.WriteByte(m.actionSpeed)
		return w.Bytes()
	}
}

func (m *SkillPrepareForeign) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.level = r.ReadByte()
		if skillPrepareForeignActionIsByte(ctx) {
			m.action = uint16(r.ReadByte())
		} else {
			m.action = r.ReadUint16()
		}
		m.actionSpeed = r.ReadByte()
	}
}
