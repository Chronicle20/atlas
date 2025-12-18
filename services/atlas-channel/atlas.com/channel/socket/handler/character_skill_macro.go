package handler

import (
	"atlas-channel/macro"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterSkillMacroHandle = "CharacterSkillMacroHandle"

func CharacterSkillMacroHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		count := r.ReadByte()
		macros := make([]macro.Model, 0)
		for i := range count {
			name := r.ReadAsciiString()
			shout := !r.ReadBool()
			skillId1 := r.ReadUint32()
			skillId2 := r.ReadUint32()
			skillId3 := r.ReadUint32()
			m := macro.NewModel(uint32(i), name, shout, skill.Id(skillId1), skill.Id(skillId2), skill.Id(skillId3))
			macros = append(macros, m)
		}
		l.Debugf("Setting [%d] skill macros for character [%d].", count, s.CharacterId())
		err := macro.NewProcessor(l, ctx).Update(s.CharacterId(), macros)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", s.CharacterId())
		}
	}
}
