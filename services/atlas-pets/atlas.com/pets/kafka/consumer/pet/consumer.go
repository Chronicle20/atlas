package pet

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("pet_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("pet_movement_command")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDespawnCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAttemptCommandCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardClosenessCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardFullnessCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardLevelCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSetExcludeCommand(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand(db))))
		}
	}
}

func handleSpawnCommand(db *gorm.DB) message.Handler[command[spawnCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[spawnCommandBody]) {
		if c.Type != CommandPetSpawn {
			return
		}
		err := pet.Spawn(l)(ctx)(db)(c.PetId, c.ActorId, c.Body.Lead)
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn pet [%d] for character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleDespawnCommand(db *gorm.DB) message.Handler[command[despawnCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[despawnCommandBody]) {
		if c.Type != CommandPetDespawn {
			return
		}
		err := pet.Despawn(l)(ctx)(db)(c.PetId, c.ActorId, "NORMAL")
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn pet [%d] for character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleAttemptCommandCommand(db *gorm.DB) message.Handler[command[attemptCommandCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[attemptCommandCommandBody]) {
		if c.Type != CommandPetAttemptCommand {
			return
		}
		err := pet.AttemptCommand(l)(ctx)(db)(c.PetId, c.ActorId, c.Body.CommandId, c.Body.ByName)
		if err != nil {
			l.WithError(err).Errorf("Unable to attempt command for pet [%d] by character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleAwardClosenessCommand(db *gorm.DB) message.Handler[command[awardClosenessCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[awardClosenessCommandBody]) {
		if c.Type != CommandAwardCloseness {
			return
		}
		_ = pet.AwardCloseness(l)(ctx)(db)(c.PetId, c.Body.Amount, c.ActorId)
	}
}

func handleAwardFullnessCommand(db *gorm.DB) message.Handler[command[awardFullnessCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[awardFullnessCommandBody]) {
		if c.Type != CommandAwardFullness {
			return
		}
		_ = pet.AwardFullness(l)(ctx)(db)(c.PetId, c.Body.Amount, c.ActorId)
	}
}

func handleAwardLevelCommand(db *gorm.DB) message.Handler[command[awardLevelCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[awardLevelCommandBody]) {
		if c.Type != CommandAwardLevel {
			return
		}
		_ = pet.AwardLevel(l)(ctx)(db)(c.PetId, c.Body.Amount, c.ActorId)
	}
}

func handleSetExcludeCommand(db *gorm.DB) message.Handler[command[setExcludeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[setExcludeCommandBody]) {
		if c.Type != CommandSetExclude {
			return
		}
		_ = pet.SetExclude(l)(ctx)(db)(c.PetId, c.Body.Items)
	}
}

func handleMovementCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
	return func(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
		m := _map.NewModel(world.Id(c.WorldId))(channel.Id(c.ChannelId))(_map.Id(c.MapId))
		err := pet.Move(l)(ctx)(db)(c.ObjectId, m, c.ObserverId, c.X, c.Y, c.Stance)
		if err != nil {
			l.WithError(err).Errorf("Error processing movement for pet [%d].", c.ObjectId)
		}
	}
}
