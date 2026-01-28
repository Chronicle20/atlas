package handler

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/skill"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterUseSkillHandle = "CharacterUseSkillHandle"

func CharacterUseSkillHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		sui := &model.SkillUsageInfo{}
		sui.Decode(l, t, readerOptions)(r)

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempting to use skill when dead.", s.CharacterId())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}

		var sm skill2.Model
		for _, rs := range c.Skills() {
			if rs.Id() == skill.Id(sui.SkillId()) {
				sm = rs
			}
		}
		if sm.Id() == 0 || sm.Level() == 0 || sm.Level() != sui.SkillLevel() {
			l.Debugf("Character [%d] attempting to use skill [%d] at level [%d], but they do not have it.", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		se, err := skill3.NewProcessor(l, ctx).GetEffect(sui.SkillId(), sui.SkillLevel())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
			return
		}

		l.Debugf("Character [%d] using skill [%d] at level [%d].", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
		err = handler.UseSkill(l)(ctx)(s.Map(), s.CharacterId(), *sui, se)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] failed to use skill [%d].", s.CharacterId(), sui.SkillId())
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(s.CharacterId(), announceSkillUse(l)(ctx)(wp)(sui.SkillId(), c.Level(), sui.SkillLevel()))

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Map(), s.CharacterId(), announceForeignSkillUse(l)(ctx)(wp)(s.CharacterId(), sui.SkillId(), c.Level(), sui.SkillLevel()))

		err = enableActions(l)(ctx)(wp)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
		}
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model.StatUpdate, 0), true))
		}
	}
}

func announceSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterSkillUseEffectBody(l)(skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}

func announceForeignSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterSkillUseEffectForeignBody(l)(characterId, skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}
