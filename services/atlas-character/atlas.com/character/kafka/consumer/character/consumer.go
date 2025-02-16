package character

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_event_status")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_movement_command")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMap(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeJob(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardExperience(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardLevel(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeMeso(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDropMeso(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeFame(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDistributeAp(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDistributeSp(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeHP(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMP(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent)))
			t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLevelChangedStatusEvent(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJobChangedStatusEvent(db))))
		}
	}
}

func handleChangeMap(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[changeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeMapBody]) {
		if c.Type != CommandChangeMap {
			return
		}

		err := character.ChangeMap(l, db, ctx)(c.CharacterId, c.WorldId, c.Body.ChannelId, c.Body.MapId, c.Body.PortalId)
		if err != nil {
			l.WithError(err).Errorf("Unable to change character [%d] map.", c.CharacterId)
		}
	}
}

func handleChangeJob(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[changeJobCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeJobCommandBody]) {
		if c.Type != CommandChangeJob {
			return
		}

		_ = character.ChangeJob(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.ChannelId, c.Body.JobId)
	}
}

func handleAwardExperience(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[awardExperienceCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[awardExperienceCommandBody]) {
		if c.Type != CommandAwardExperience {
			return
		}

		es, err := model.SliceMap(func(m experienceDistributions) (character.ExperienceModel, error) {
			return character.NewExperienceModel(m.ExperienceType, m.Amount, m.Attr1), nil
		})(model.FixedProvider(c.Body.Distributions))()()
		if err != nil {
			return
		}

		_ = character.AwardExperience(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.ChannelId, es)
	}
}

func handleAwardLevel(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[awardLevelCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[awardLevelCommandBody]) {
		if c.Type != CommandAwardLevel {
			return
		}

		_ = character.AwardLevel(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.ChannelId, c.Body.Amount)
	}
}

func handleRequestChangeMeso(db *gorm.DB) message.Handler[command[requestChangeMesoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestChangeMesoBody]) {
		if c.Type != CommandRequestChangeMeso {
			return
		}

		_ = character.RequestChangeMeso(l)(ctx)(db)(c.CharacterId, c.Body.Amount, c.Body.ActorId, c.Body.ActorType)
	}
}

func handleRequestDropMeso(db *gorm.DB) message.Handler[command[requestDropMesoCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDropMesoCommandBody]) {
		if c.Type != CommandRequestDropMeso {
			return
		}

		_ = character.RequestDropMeso(l)(ctx)(db)(c.WorldId, c.Body.ChannelId, c.Body.MapId, c.CharacterId, c.Body.Amount)
	}
}

func handleRequestChangeFame(db *gorm.DB) message.Handler[command[requestChangeFameBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestChangeFameBody]) {
		if c.Type != CommandRequestChangeFame {
			return
		}

		_ = character.RequestChangeFame(l)(ctx)(db)(c.CharacterId, c.Body.Amount, c.Body.ActorId, c.Body.ActorType)
	}
}

func handleRequestDistributeAp(db *gorm.DB) message.Handler[command[requestDistributeApCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDistributeApCommandBody]) {
		if c.Type != CommandRequestDistributeAp {
			return
		}

		dp := model.SliceMap(transform)(model.FixedProvider(c.Body.Distributions))()
		ds, err := model.FilteredProvider(dp, model.Filters[character.Distribution](func(d character.Distribution) bool {
			return d.Amount > 0
		}))()
		if err != nil {
			return
		}
		_ = character.RequestDistributeAp(l)(ctx)(db)(c.CharacterId, ds)
	}
}

func handleRequestDistributeSp(db *gorm.DB) message.Handler[command[requestDistributeSpCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDistributeSpCommandBody]) {
		if c.Type != CommandRequestDistributeSp {
			return
		}
		_ = character.RequestDistributeSp(l)(ctx)(db)(c.CharacterId, c.Body.SkillId, c.Body.Amount)
	}
}

func transform(m DistributePair) (character.Distribution, error) {
	return character.Distribution{
		Ability: m.Ability,
		Amount:  m.Amount,
	}, nil
}

func handleChangeHP(db *gorm.DB) message.Handler[command[changeHPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeHPBody]) {
		if c.Type != CommandChangeHP {
			return
		}
		_ = character.ChangeHP(l)(ctx)(db)(c.WorldId, c.Body.ChannelId, c.CharacterId, c.Body.Amount)
	}
}

func handleChangeMP(db *gorm.DB) message.Handler[command[changeMPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeMPBody]) {
		if c.Type != CommandChangeMP {
			return
		}
		_ = character.ChangeMP(l)(ctx)(db)(c.WorldId, c.Body.ChannelId, c.CharacterId, c.Body.Amount)
	}
}

func handleLevelChangedStatusEvent(db *gorm.DB) message.Handler[statusEvent[levelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[levelChangedStatusEventBody]) {
		if e.Type != StatusEventTypeLevelChanged {
			return
		}
		_ = character.ProcessLevelChange(l)(ctx)(db)(e.WorldId, e.Body.ChannelId, e.CharacterId, e.Body.Amount)
	}
}

func handleJobChangedStatusEvent(db *gorm.DB) message.Handler[statusEvent[jobChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[jobChangedStatusEventBody]) {
		if e.Type != StatusEventTypeJobChanged {
			return
		}
		_ = character.ProcessJobChange(l)(ctx)(db)(e.WorldId, e.Body.ChannelId, e.CharacterId, e.Body.JobId)
	}
}

func handleMovementEvent(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
	err := character.Move(l)(ctx)(c.CharacterId)(c.WorldId)(c.ChannelId)(c.MapId)(c.Movement)
	if err != nil {
		l.WithError(err).Errorf("Error processing movement for character [%d].", c.CharacterId)
	}
}
