package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterSkillMacroWriter = "CharacterSkillMacro"

// SkillMacroEntry is one row of the client's Skill-window macro list: a name,
// the "shout the macro name in chat" flag, and up to three bound skill ids.
type SkillMacroEntry struct {
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewSkillMacroEntry(name string, shout bool, skillId1 skill.Id, skillId2 skill.Id, skillId3 skill.Id) SkillMacroEntry {
	return SkillMacroEntry{name: name, shout: shout, skillId1: skillId1, skillId2: skillId2, skillId3: skillId3}
}

func (e SkillMacroEntry) Name() string       { return e.name }
func (e SkillMacroEntry) Shout() bool        { return e.shout }
func (e SkillMacroEntry) SkillId1() skill.Id { return e.skillId1 }
func (e SkillMacroEntry) SkillId2() skill.Id { return e.skillId2 }
func (e SkillMacroEntry) SkillId3() skill.Id { return e.skillId3 }

// SkillMacro is the clientbound MACRO_SYS_DATA_INIT packet
// (CWvsContext::OnMacroSysDataInit): the server hands the client the character's
// whole macro list, at login and after every macro update.
//
// Byte layout: see docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md.
// The layout is byte-identical across every populated version (gms_v61..jms_v185;
// gms_v48 has no macro feature at all) — no version gate.
//
// Shout polarity is INVERTED on the wire: wire byte 0 means "shout on". Settled
// on gms_v83's CMacroSysMan::IsShoutMacro (0x631d19, `field25 == 0` drives the
// shout checkbox's checked state) and independently confirmed by gms_v95's
// SINGLEMACRO::Decode/Encode (0x4f97f0/0x4f9710), where the field is literally
// named bMute and stored/encoded with no inversion (bMute==negation of shout).
// Both Encode and Decode below invert the in-memory bool at the wire boundary
// so callers work with an upright "shout" meaning; see layout-derivation.md
// §"Shout polarity" for the full per-version evidence.
//
// packet-audit:fname CWvsContext::OnMacroSysDataInit
type SkillMacro struct {
	macros []SkillMacroEntry
}

func NewSkillMacro(macros ...SkillMacroEntry) SkillMacro {
	return SkillMacro{macros: macros}
}

func (m SkillMacro) Macros() []SkillMacroEntry { return m.macros }

func (m SkillMacro) Operation() string { return CharacterSkillMacroWriter }

func (m SkillMacro) String() string {
	return fmt.Sprintf("macros [%d]", len(m.macros))
}

func (m SkillMacro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.macros)))
		for _, e := range m.macros {
			w.WriteAsciiString(e.name)
			// INVERTED: wire 0 = shout on. See struct doc comment.
			w.WriteBool(!e.shout)
			w.WriteInt(uint32(e.skillId1))
			w.WriteInt(uint32(e.skillId2))
			w.WriteInt(uint32(e.skillId3))
		}
		return w.Bytes()
	}
}

func (m *SkillMacro) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.macros = make([]SkillMacroEntry, 0, count)
		for range count {
			name := r.ReadAsciiString()
			// INVERTED: wire 0 = shout on. See struct doc comment.
			shout := !r.ReadBool()
			skillId1 := skill.Id(r.ReadUint32())
			skillId2 := skill.Id(r.ReadUint32())
			skillId3 := skill.Id(r.ReadUint32())
			m.macros = append(m.macros, NewSkillMacroEntry(name, shout, skillId1, skillId2, skillId3))
		}
	}
}
