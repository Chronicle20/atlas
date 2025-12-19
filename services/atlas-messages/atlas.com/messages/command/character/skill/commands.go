package skill

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/skill"
	"atlas-messages/saga"
	skill3 "atlas-messages/skill"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"time"
)

func MaxSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sdp := skill.NewProcessor(l, ctx)
		sagaProcessor := saga.NewProcessor(l, ctx)
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

					return sagaProcessor.Create(sagaBuilder.Build())
				}
			}, true
		}
	}
}

func ResetSkillCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sdp := skill.NewProcessor(l, ctx)
		sagaProcessor := saga.NewProcessor(l, ctx)
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

					return sagaProcessor.Create(sagaBuilder.Build())
				}
			}, true
		}
	}
}
