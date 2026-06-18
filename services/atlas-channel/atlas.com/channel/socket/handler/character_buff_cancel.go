package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/door"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	skill "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterBuffCancelHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.BuffCancelRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = buff.NewProcessor(l, ctx).Cancel(s.Field(), s.CharacterId(), p.SkillId())
		// Cancelling the Mystic Door buff dismisses the door early.
		if p.SkillId() == int32(skill.PriestMysticDoorId) {
			_ = door.NewProcessor(l, ctx).Remove(s.Field(), s.CharacterId(), "CANCELLED")
		}
	}
}
