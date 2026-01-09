package storage

import (
	consumer2 "atlas-channel/kafka/consumer"
	storage2 "atlas-channel/kafka/message/storage"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("storage_show_command")(storage2.EnvShowStorageCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("storage_status_event")(storage2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(storage2.EnvShowStorageCommandTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleShowStorageCommand(sc, wp))))

				t, _ = topic.EnvProvider(l)(storage2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMesosUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleArrangedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStorageErrorEvent(sc, wp))))
			}
		}
	}
}

func handleShowStorageCommand(sc server.Model, wp writer.Producer) message.Handler[storage2.ShowStorageCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd storage2.ShowStorageCommand) {
		if cmd.Type != storage2.CommandTypeShowStorage {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, world.Id(cmd.WorldId), channel.Id(cmd.ChannelId)) {
			return
		}

		l.Debugf("Received ShowStorage command for character %d, NPC %d, account %d", cmd.CharacterId, cmd.NpcId, cmd.AccountId)

		// Fetch storage data from storage service
		storageProc := storage.NewProcessor(l, ctx)
		storageData, err := storageProc.GetStorageData(cmd.AccountId, cmd.WorldId)
		if err != nil {
			l.WithError(err).Errorf("Unable to fetch storage data for account %d.", cmd.AccountId)
			return
		}

		// Send storage UI to the character and set the storage NPC ID on the session
		sessionProc := session.NewProcessor(l, ctx)
		err = sessionProc.IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(cmd.CharacterId, func(s session.Model) error {
			// Store the NPC ID for fee lookup during storage operations
			sessionProc.SetStorageNpcId(s.SessionId(), cmd.NpcId)

			return session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationShowBody(l, sc.Tenant())(cmd.NpcId, storageData.Capacity, storageData.Mesos, storageData.Assets))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to show storage to character [%d].", cmd.CharacterId)
		}
	}
}

func handleMesosUpdatedEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StatusEvent[storage2.MesosUpdatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StatusEvent[storage2.MesosUpdatedEventBody]) {
		if e.Type != storage2.StatusEventTypeMesosUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.WorldId != byte(sc.WorldId()) {
			return
		}

		l.Debugf("Received MesosUpdated event for account [%d]. Old: [%d], New: [%d]", e.AccountId, e.Body.OldMesos, e.Body.NewMesos)

		// Find the session by account and send the meso update
		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.WorldId(), sc.ChannelId())(e.AccountId,
			session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationUpdateMesoBody(l)(0, e.Body.NewMesos)))
		if err != nil {
			l.WithError(err).Errorf("Unable to send meso update to account [%d].", e.AccountId)
		}
	}
}

func handleArrangedEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StatusEvent[storage2.ArrangedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StatusEvent[storage2.ArrangedEventBody]) {
		if e.Type != storage2.StatusEventTypeArranged {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.WorldId != byte(sc.WorldId()) {
			return
		}

		l.Debugf("Received Arranged event for account [%d].", e.AccountId)

		// Fetch the updated storage data and resend to client
		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.WorldId(), sc.ChannelId())(e.AccountId, func(s session.Model) error {
			storageData, err := storage.NewProcessor(l, ctx).GetStorageData(e.AccountId, e.WorldId)
			if err != nil {
				return err
			}

			// Send refreshed storage view
			return session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationShowBody(l, sc.Tenant())(0, storageData.Capacity, storageData.Mesos, storageData.Assets))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to refresh storage for account [%d].", e.AccountId)
		}
	}
}

func handleStorageErrorEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StatusEvent[storage2.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StatusEvent[storage2.ErrorEventBody]) {
		if e.Type != storage2.StatusEventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.WorldId != byte(sc.WorldId()) {
			return
		}

		l.Debugf("Received storage error event for account [%d]. Code: [%s], Message: [%s]", e.AccountId, e.Body.ErrorCode, e.Body.Message)

		// Map error code to appropriate writer
		var writerBody writer.BodyProducer
		switch e.Body.ErrorCode {
		case storage2.ErrorCodeStorageFull:
			writerBody = writer.StorageOperationErrorInventoryFullBody(l)
		case storage2.ErrorCodeNotEnoughMesos:
			writerBody = writer.StorageOperationErrorNotEnoughMesoBody(l)
		case storage2.ErrorCodeOneOfAKind:
			writerBody = writer.StorageOperationErrorOneOfAKindBody(l)
		default:
			writerBody = writer.StorageOperationErrorMessageBody(l)(e.Body.Message)
		}

		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.WorldId(), sc.ChannelId())(e.AccountId,
			session.Announce(l)(ctx)(wp)(writer.StorageOperation)(writerBody))
		if err != nil {
			l.WithError(err).Errorf("Unable to send error to account [%d].", e.AccountId)
		}
	}
}
