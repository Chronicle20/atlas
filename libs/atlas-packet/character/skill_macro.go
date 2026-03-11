package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillMacroHandle = "CharacterSkillMacroHandle"
const CharacterSkillMacroWriter = "CharacterSkillMacro"

type SkillMacroEntry struct {
	Name     string
	Shout    bool
	SkillId1 uint32
	SkillId2 uint32
	SkillId3 uint32
}

// SkillMacro - CUser::SendSkillMacroModifiedMessage
type SkillMacro struct {
	macros []SkillMacroEntry
}

func (m SkillMacro) Macros() []SkillMacroEntry { return m.macros }

func (m SkillMacro) Operation() string {
	return CharacterSkillMacroHandle
}

func (m SkillMacro) String() string {
	return fmt.Sprintf("macros [%d]", len(m.macros))
}

func (m SkillMacro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.macros)))
		for _, e := range m.macros {
			w.WriteAsciiString(e.Name)
			w.WriteBool(!e.Shout)
			w.WriteInt(e.SkillId1)
			w.WriteInt(e.SkillId2)
			w.WriteInt(e.SkillId3)
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
			shout := !r.ReadBool()
			skillId1 := r.ReadUint32()
			skillId2 := r.ReadUint32()
			skillId3 := r.ReadUint32()
			m.macros = append(m.macros, SkillMacroEntry{
				Name:     name,
				Shout:    shout,
				SkillId1: skillId1,
				SkillId2: skillId2,
				SkillId3: skillId3,
			})
		}
	}
}
