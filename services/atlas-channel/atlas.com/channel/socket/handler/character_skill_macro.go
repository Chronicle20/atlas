package handler

import (
	"atlas-channel/macro"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterSkillMacroHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.SkillMacro{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		macros := make([]macro.Model, 0)
		for i, e := range p.Macros() {
			m := macro.NewModel(uint32(i), e.Name, e.Shout, skill.Id(e.SkillId1), skill.Id(e.SkillId2), skill.Id(e.SkillId3))
			macros = append(macros, m)
		}
		err := macro.NewProcessor(l, ctx).Update(s.CharacterId(), macros)
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", s.CharacterId())
		}
	}
}
