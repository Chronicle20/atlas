package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterBuffCancelHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.BuffCancelRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = buff.NewProcessor(l, ctx).Cancel(s.Field(), s.CharacterId(), p.SkillId())

		skillId := uint32(p.SkillId())
		if skill.IsKeyDownSkill(skill.Id(skillId)) {
			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
			if err == nil && shouldBroadcastKeydown(c.Skills(), skillId) {
				_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(),
					AnnounceForeignSkillCancel(l)(ctx)(wp)(s.CharacterId(), skillId))
			}
		}
	}
}
