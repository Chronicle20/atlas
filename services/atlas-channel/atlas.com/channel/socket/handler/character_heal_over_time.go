package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterHealOverTimeHandle = "CharacterHealOverTimeHandle"

func CharacterHealOverTimeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		val := r.ReadUint32()
		hp := r.ReadInt16()
		mp := r.ReadInt16()
		b := byte(0)
		if t.Region() == "GMS" && t.MajorVersion() <= 95 {
			b = r.ReadByte()
		}
		l.Debugf("Character [%d] received [%d] HP and [%d] MP over time. updateTime [%d], 5120 [%d], b [%d]", s.CharacterId(), hp, mp, updateTime, val, b)
		if hp != 0 {
			_ = character.NewProcessor(l, ctx).ChangeHP(s.Field(), s.CharacterId(), hp)
		}
		if mp != 0 {
			_ = character.NewProcessor(l, ctx).ChangeMP(s.Field(), s.CharacterId(), mp)
		}
	}
}
