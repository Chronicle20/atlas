package storage

import (
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message"
	"atlas-storage/projection"
	"atlas-storage/storage"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("storage_command")(message.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(message.EnvCommandTopic)()
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleDepositCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleWithdrawCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleUpdateMesosCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleDepositRollbackCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleArrangeCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleShowStorageCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleCloseStorageCommand())))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleExpireCommand(db))))
		}
	}
}

func handleDepositCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.DepositBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.DepositBody]) {
		if c.Type != message.CommandTypeDeposit {
			return
		}

		_, err := storage.NewProcessor(l, ctx, db).DepositAndEmit(c.TransactionId, c.WorldId, c.AccountId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to deposit item [%d] for account [%d] world [%d].", c.Body.TemplateId, c.AccountId, c.WorldId)
		}
	}
}

func handleWithdrawCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.WithdrawBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.WithdrawBody]) {
		if c.Type != message.CommandTypeWithdraw {
			return
		}

		err := storage.NewProcessor(l, ctx, db).WithdrawAndEmit(c.TransactionId, c.WorldId, c.AccountId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to withdraw asset [%d] for account [%d] world [%d].", c.Body.AssetId, c.AccountId, c.WorldId)
		}
	}
}

func handleUpdateMesosCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.UpdateMesosBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.UpdateMesosBody]) {
		if c.Type != message.CommandTypeUpdateMesos {
			return
		}

		err := storage.NewProcessor(l, ctx, db).UpdateMesosAndEmit(c.TransactionId, c.WorldId, c.AccountId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to update mesos for account [%d] world [%d].", c.AccountId, c.WorldId)
		}
	}
}

func handleDepositRollbackCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.DepositRollbackBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.DepositRollbackBody]) {
		if c.Type != message.CommandTypeDepositRollback {
			return
		}

		err := storage.NewProcessor(l, ctx, db).DepositRollback(c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to rollback deposit for asset [%d] account [%d] world [%d].", c.Body.AssetId, c.AccountId, c.WorldId)
		}
	}
}

func handleArrangeCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.ArrangeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.ArrangeBody]) {
		if c.Type != message.CommandTypeArrange {
			return
		}

		err := storage.NewProcessor(l, ctx, db).ArrangeAndEmit(c.TransactionId, c.WorldId, c.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Unable to arrange storage for account [%d] world [%d].", c.AccountId, c.WorldId)
		}
	}
}

func handleShowStorageCommand(db *gorm.DB) kafkaMessage.Handler[message.ShowStorageCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.ShowStorageCommand) {
		if c.Type != message.CommandTypeShowStorage {
			return
		}

		l.Debugf("Received ShowStorage command for character [%d], NPC [%d], account [%d], world [%d]",
			c.CharacterId, c.NpcId, c.AccountId, c.WorldId)

		t := tenant.MustFromContext(ctx)

		// Build the projection from storage data
		proj, err := projection.BuildProjection(l, db, t.Id())(c.CharacterId, c.AccountId, c.WorldId, c.NpcId)
		if err != nil {
			l.WithError(err).Errorf("Failed to build projection for character [%d], account [%d], world [%d]",
				c.CharacterId, c.AccountId, c.WorldId)
			return
		}

		// Store projection in manager
		projection.GetManager().Create(c.CharacterId, proj)

		l.Debugf("Created projection for character [%d] with storage [%s], capacity [%d], mesos [%d]",
			c.CharacterId, proj.StorageId(), proj.Capacity(), proj.Mesos())

		// Emit PROJECTION_CREATED event
		ch := channel.NewModel(c.WorldId, c.ChannelId)
		err = storage.NewProcessor(l, ctx, db).EmitProjectionCreatedEvent(c.CharacterId, c.AccountId, ch, c.NpcId)
		if err != nil {
			l.WithError(err).Errorf("Failed to emit PROJECTION_CREATED event for character [%d]", c.CharacterId)
		}
	}
}

func handleCloseStorageCommand() kafkaMessage.Handler[message.CloseStorageCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.CloseStorageCommand) {
		if c.Type != message.CommandTypeCloseStorage {
			return
		}

		l.Debugf("Received CloseStorage command for character [%d]", c.CharacterId)

		// Remove NPC context from cache (legacy)
		cache := storage.GetNpcContextCache()
		cache.Remove(c.CharacterId)

		// Delete the projection
		projection.GetManager().Delete(c.CharacterId)

		l.Debugf("Removed projection and NPC context for character [%d]", c.CharacterId)
	}
}

func handleExpireCommand(db *gorm.DB) kafkaMessage.Handler[message.Command[message.ExpireBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c message.Command[message.ExpireBody]) {
		if c.Type != message.CommandTypeExpire {
			return
		}

		l.Debugf("Received EXPIRE command for account [%d], asset [%d], template [%d].",
			c.AccountId, c.Body.AssetId, c.Body.TemplateId)

		// Storage items are always non-cash items (cash items are in cashshop)
		isCash := false

		err := storage.NewProcessor(l, ctx, db).ExpireAndEmit(
			c.TransactionId,
			c.WorldId,
			c.AccountId,
			uint32(c.Body.AssetId),
			isCash,
			c.Body.ReplaceItemId,
			c.Body.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire storage asset [%d] for account [%d].", c.Body.AssetId, c.AccountId)
		}
	}
}
