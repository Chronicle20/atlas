package skill

import (
	"atlas-messages/character"
	skill2 "atlas-messages/character/skill"
	"atlas-messages/command"
	"atlas-messages/skill"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"time"
)

func MaxSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@skill\s+max\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 0 {
				return nil, false
			}
			skillString := match[1]
			skillId, err := strconv.Atoi(skillString)
			if err != nil {
				return nil, false
			}

			si, err := skill.GetById(l)(ctx)(uint32(skillId))
			if err != nil {
				return nil, false
			}
			masterLevel := byte(len(si.Effects()))

			decs := model.Decorators[character.Model](character.SkillModelDecorator(l)(ctx))
			sc, err := model.Map(model.Decorate(decs))(model.FixedProvider(c))()
			if err != nil {
				return nil, false
			}
			var s *skill2.Model
			for _, rs := range sc.Skills() {
				if rs.Id() == uint32(skillId) {
					s = &rs
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					if s == nil {
						return skill2.RequestCreate(l)(ctx)(c.Id(), uint32(skillId), masterLevel, masterLevel, time.Time{})
					} else {
						return skill2.RequestUpdate(l)(ctx)(c.Id(), uint32(skillId), masterLevel, masterLevel, time.Time{})
					}
				}
			}, true
		}
	}
}

func ResetSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@skill\s+reset\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 0 {
				return nil, false
			}
			skillString := match[1]
			skillId, err := strconv.Atoi(skillString)
			if err != nil {
				return nil, false
			}

			si, err := skill.GetById(l)(ctx)(uint32(skillId))
			if err != nil {
				return nil, false
			}
			masterLevel := byte(len(si.Effects()))

			decs := model.Decorators[character.Model](character.SkillModelDecorator(l)(ctx))
			sc, err := model.Map(model.Decorate(decs))(model.FixedProvider(c))()
			if err != nil {
				return nil, false
			}
			var s *skill2.Model
			for _, rs := range sc.Skills() {
				if rs.Id() == uint32(skillId) {
					s = &rs
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					if s == nil {
						return skill2.RequestCreate(l)(ctx)(c.Id(), uint32(skillId), 0, masterLevel, time.Time{})
					} else {
						return skill2.RequestUpdate(l)(ctx)(c.Id(), uint32(skillId), 0, masterLevel, time.Time{})
					}
				}
			}, true
		}
	}
}
