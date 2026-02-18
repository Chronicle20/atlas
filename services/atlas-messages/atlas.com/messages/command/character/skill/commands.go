package skill

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/skill"
	"atlas-messages/saga"
	skill3 "atlas-messages/skill"
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func MaxSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sdp := skill.NewProcessor(l, ctx)
		sagaProcessor := saga.NewProcessor(l, ctx)
		return func(_ field.Model, c character.Model, m string) (command.Executor, bool) {
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

			si, err := sdp.GetById(uint32(skillId))
			if err != nil {
				return nil, false
			}
			masterLevel := byte(len(si.Effects()))

			decs := model.Decorators[character.Model](cp.SkillModelDecorator)
			sc, err := model.Map(model.Decorate(decs))(model.FixedProvider(c))()
			if err != nil {
				return nil, false
			}
			var s *skill3.Model
			for _, rs := range sc.Skills() {
				if rs.Id() == uint32(skillId) {
					s = &rs
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					sagaBuilder := saga.NewBuilder().
						SetSagaType(saga.QuestReward).
						SetInitiatedBy("COMMAND")

					if s == nil {
						sagaBuilder.AddStep(
							"create_skill",
							saga.Pending,
							saga.CreateSkill,
							saga.CreateSkillPayload{
								CharacterId: c.Id(),
								SkillId:     uint32(skillId),
								Level:       masterLevel,
								MasterLevel: masterLevel,
								Expiration:  time.Time{},
							},
						)
					} else {
						sagaBuilder.AddStep(
							"update_skill",
							saga.Pending,
							saga.UpdateSkill,
							saga.UpdateSkillPayload{
								CharacterId: c.Id(),
								SkillId:     uint32(skillId),
								Level:       masterLevel,
								MasterLevel: masterLevel,
								Expiration:  time.Time{},
							},
						)
					}

					s, err := sagaBuilder.Build()
					if err != nil {
						return err
					}
					return sagaProcessor.Create(s)
				}
			}, true
		}
	}
}

func ResetSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sdp := skill.NewProcessor(l, ctx)
		sagaProcessor := saga.NewProcessor(l, ctx)
		return func(_ field.Model, c character.Model, m string) (command.Executor, bool) {
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

			si, err := sdp.GetById(uint32(skillId))
			if err != nil {
				return nil, false
			}
			masterLevel := byte(len(si.Effects()))

			decs := model.Decorators[character.Model](cp.SkillModelDecorator)
			sc, err := model.Map(model.Decorate(decs))(model.FixedProvider(c))()
			if err != nil {
				return nil, false
			}
			var s *skill3.Model
			for _, rs := range sc.Skills() {
				if rs.Id() == uint32(skillId) {
					s = &rs
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					sagaBuilder := saga.NewBuilder().
						SetSagaType(saga.QuestReward).
						SetInitiatedBy("COMMAND")

					if s == nil {
						sagaBuilder.AddStep(
							"create_skill",
							saga.Pending,
							saga.CreateSkill,
							saga.CreateSkillPayload{
								CharacterId: c.Id(),
								SkillId:     uint32(skillId),
								Level:       0,
								MasterLevel: masterLevel,
								Expiration:  time.Time{},
							},
						)
					} else {
						sagaBuilder.AddStep(
							"update_skill",
							saga.Pending,
							saga.UpdateSkill,
							saga.UpdateSkillPayload{
								CharacterId: c.Id(),
								SkillId:     uint32(skillId),
								Level:       0,
								MasterLevel: masterLevel,
								Expiration:  time.Time{},
							},
						)
					}

					s, err := sagaBuilder.Build()
					if err != nil {
						return err
					}
					return sagaProcessor.Create(s)
				}
			}, true
		}
	}
}
