package writer

import (
	"atlas-channel/macro"
	"github.com/Chronicle20/atlas-socket/response"
)

const (
	CharacterSkillMacro = "CharacterSkillMacro"
)

func CharacterSkillMacroBody(macros []macro.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(len(macros)))
		for _, m := range macros {
			w.WriteAsciiString(m.Name())
			w.WriteBool(m.Shout())
			w.WriteInt(uint32(m.SkillId1()))
			w.WriteInt(uint32(m.SkillId2()))
			w.WriteInt(uint32(m.SkillId3()))
		}
		return w.Bytes()
	}
}
