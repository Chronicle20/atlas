package pet

import (
	consumer2 "atlas-pets/kafka/consumer"
	pet2 "atlas-pets/kafka/message/pet"
	"atlas-pets/pet"
	"context"

	"github.com/Chronicle20/atlas-constants/field"

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
			rf(consumer2.NewConfig(l)("pet_command")(pet2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("pet_movement_command")(pet2.EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(pet2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDespawnCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAttemptCommandCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardClosenessCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardFullnessCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardLevelCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSetExcludeCommand(db))))
			t, _ = topic.EnvProvider(l)(pet2.EnvCommandTopicMovement)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand(db))))
		}
	}
}

func handleSpawnCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.SpawnCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.SpawnCommandBody]) {
		if c.Type != pet2.CommandPetSpawn {
			return
		}
		err := pet.NewProcessor(l, ctx, db).SpawnAndEmit(c.PetId, c.ActorId, c.Body.Lead)
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn pet [%d] for character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleDespawnCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.DespawnCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.DespawnCommandBody]) {
		if c.Type != pet2.CommandPetDespawn {
			return
		}
		err := pet.NewProcessor(l, ctx, db).DespawnAndEmit(c.PetId, c.ActorId, "NORMAL")
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn pet [%d] for character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleAttemptCommandCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.AttemptCommandCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.AttemptCommandCommandBody]) {
		if c.Type != pet2.CommandPetAttemptCommand {
			return
		}
		err := pet.NewProcessor(l, ctx, db).AttemptCommandAndEmit(c.PetId, c.ActorId, c.Body.CommandId)
		if err != nil {
			l.WithError(err).Errorf("Unable to attempt command for pet [%d] by character [%d].", c.PetId, c.ActorId)
		}
	}
}

func handleAwardClosenessCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.AwardClosenessCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.AwardClosenessCommandBody]) {
		if c.Type != pet2.CommandAwardCloseness {
			return
		}
		_ = pet.NewProcessor(l, ctx, db).AwardClosenessWithTransactionAndEmit(c.TransactionId, c.PetId, c.Body.Amount)
	}
}

func handleAwardFullnessCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.AwardFullnessCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.AwardFullnessCommandBody]) {
		if c.Type != pet2.CommandAwardFullness {
			return
		}
		_ = pet.NewProcessor(l, ctx, db).AwardFullnessAndEmit(c.PetId, c.Body.Amount)
	}
}

func handleAwardLevelCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.AwardLevelCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.AwardLevelCommandBody]) {
		if c.Type != pet2.CommandAwardLevel {
			return
		}
		_ = pet.NewProcessor(l, ctx, db).AwardLevelAndEmit(c.PetId, c.Body.Amount)
	}
}

func handleSetExcludeCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.SetExcludeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.SetExcludeCommandBody]) {
		if c.Type != pet2.CommandSetExclude {
			return
		}
		_ = pet.NewProcessor(l, ctx, db).SetExcludeAndEmit(c.PetId, c.Body.Items)
	}
}

func handleMovementCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c pet2.MovementCommand) {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.MovementCommand) {
		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		err := pet.NewProcessor(l, ctx, db).Move(uint32(c.ObjectId), f, c.ObserverId, c.X, c.Y, c.Stance)
		if err != nil {
			l.WithError(err).Errorf("Error processing movement for pet [%d].", c.ObjectId)
		}
	}
}
