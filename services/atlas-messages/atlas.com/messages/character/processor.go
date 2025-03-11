package character

import (
	"atlas-messages/character/skill"
	"atlas-messages/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func byNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
		return func(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
			return func(name string) model.Provider[[]Model] {
				ps := requests.SliceProvider[RestModel, Model](l, ctx)(requestByName(name), Extract, model.Filters[Model]())
				return model.SliceMap(model.Decorate(decorators))(ps)(model.ParallelMap())
			}
		}
	}
}

func GetByName(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
			return func(name string) (Model, error) {
				return model.First(byNameProvider(l)(ctx)(decorators...)(name), model.Filters[Model]())
			}
		}
	}
}

func IdByNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(name string) model.Provider[uint32] {
	return func(ctx context.Context) func(name string) model.Provider[uint32] {
		return func(name string) model.Provider[uint32] {
			c, err := GetByName(l)(ctx)()(name)
			if err != nil {
				return model.ErrorProvider[uint32](err)
			}
			return model.FixedProvider(c.Id())
		}
	}
}

func SkillModelDecorator(l logrus.FieldLogger) func(ctx context.Context) model.Decorator[Model] {
	return func(ctx context.Context) model.Decorator[Model] {
		return func(m Model) Model {
			ms, err := skill.GetByCharacterId(l)(ctx)(m.Id())
			if err != nil {
				return m
			}
			return m.SetSkills(ms)
		}
	}
}

func AwardExperience(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, amount uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, amount uint32) error {
		return func(worldId byte, channelId byte, characterId uint32, amount uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardExperienceCommandProvider(characterId, worldId, channelId, amount))
		}
	}
}

func AwardLevel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, amount byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, amount byte) error {
		return func(worldId byte, channelId byte, characterId uint32, amount byte) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardLevelCommandProvider(characterId, worldId, channelId, amount))
		}
	}
}

func ChangeJob(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
		return func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeJobCommandProvider(characterId, worldId, channelId, jobId))
		}
	}
}
