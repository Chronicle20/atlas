package handler

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	"atlas-channel/session"
	skill3 "atlas-channel/skill"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"sync"
)

const CharacterUseSkillHandle = "CharacterUseSkillHandle"

var skillHandlerMap map[skill.Id]handler.Handler
var skillHandlerOnce sync.Once

func CharacterUseSkillHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		sui := &model.SkillUsageInfo{}
		sui.Decode(l, t, readerOptions)(r)

		c, err := character.GetById(l)(ctx)(character.SkillModelDecorator(l)(ctx))(s.CharacterId())
		if err != nil {
			_ = enableActions(l)(ctx)(wp)(s)
			return
		}
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempting to use skill when dead.", s.CharacterId())
			_ = enableActions(l)(ctx)(wp)(s)
			return
		}

		var sm skill2.Model
		for _, rs := range c.Skills() {
			if rs.Id() == sui.SkillId() {
				sm = rs
			}
		}
		if sm.Id() == 0 || sm.Level() == 0 || sm.Level() != sui.SkillLevel() {
			l.Debugf("Character [%d] attempting to use skill [%d] at level [%d], but they do not have it.", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		var h handler.Handler
		var ok bool
		if h, ok = GetSkillHandler(skill.Id(sui.SkillId())); !ok {
			l.Infof("Character [%d] attempting to use unhandled skill [%d].", s.CharacterId(), sui.SkillId())
			_ = enableActions(l)(ctx)(wp)(s)
			return
		}

		se, err := skill3.GetEffect(l)(ctx)(sui.SkillId(), sui.SkillLevel())
		if err != nil {
			_ = enableActions(l)(ctx)(wp)(s)
			return
		}

		l.Debugf("Character [%d] using skill [%d] at level [%d].", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
		err = h(l)(ctx)(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId(), *sui, se)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] failed to use skill [%d].", s.CharacterId(), sui.SkillId())
		}
		_ = enableActions(l)(ctx)(wp)(s)
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(writer.StatChanged)(s, writer.StatChangedBody(l)(make([]model.StatUpdate, 0), true))
				if err != nil {
					l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
					return err
				}
				return nil
			}
		}
	}
}

func GetSkillHandler(id skill.Id) (handler.Handler, bool) {
	skillHandlerOnce.Do(func() {
		skillHandlerMap = make(map[skill.Id]handler.Handler)
		skillHandlerMap[skill.AssassinHasteId] = handler.UseSkillHaste
		skillHandlerMap[skill.HermitMesoUpId] = handler.UseMesoUp
		skillHandlerMap[skill.BanditHasteId] = handler.UseSkillHaste
		skillHandlerMap[skill.NightWalkerStage2HasteId] = handler.UseSkillHaste
	})
	h, ok := skillHandlerMap[id]
	return h, ok
}
