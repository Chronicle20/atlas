package storage

import (
	"atlas-channel/asset"
	consumer2 "atlas-channel/kafka/consumer"
	storage2 "atlas-channel/kafka/message/storage"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"
	"sort"

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
			rf(consumer2.NewConfig(l)("storage_status_event")(storage2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("storage_compartment_status_event")(storage2.EnvEventTopicStorageCompartmentStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(storage2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMesosUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleArrangedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStorageErrorEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleProjectionCreatedEvent(sc, wp))))

				t, _ = topic.EnvProvider(l)(storage2.EnvEventTopicStorageCompartmentStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStorageCompartmentAcceptedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStorageCompartmentReleasedEvent(sc, wp))))
			}
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

		if e.WorldId != sc.WorldId() {
			return
		}

		l.Debugf("Received MesosUpdated event for account [%d]. Old: [%d], New: [%d]", e.AccountId, e.Body.OldMesos, e.Body.NewMesos)

		// Find the session by account and send the meso update
		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(e.AccountId,
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

		if e.WorldId != sc.WorldId() {
			return
		}

		l.Debugf("Received Arranged event for account [%d].", e.AccountId)

		// Fetch the updated storage data and resend to client
		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(e.AccountId, func(s session.Model) error {
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

		if e.WorldId != sc.WorldId() {
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

		err := session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(e.AccountId,
			session.Announce(l)(ctx)(wp)(writer.StorageOperation)(writerBody))
		if err != nil {
			l.WithError(err).Errorf("Unable to send error to account [%d].", e.AccountId)
		}
	}
}

func handleStorageCompartmentAcceptedEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StorageCompartmentEvent[storage2.CompartmentAcceptedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StorageCompartmentEvent[storage2.CompartmentAcceptedEventBody]) {
		if e.Type != storage2.StatusEventTypeCompartmentAccepted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.WorldId != sc.WorldId() {
			return
		}

		if e.CharacterId == 0 {
			l.Warnf("Received storage compartment ACCEPTED event for account [%d] but no character ID specified", e.AccountId)
			return
		}

		inventoryType := asset.InventoryType(e.Body.InventoryType)
		l.Debugf("Storage compartment ACCEPTED: character [%d] account [%d] asset [%d] slot [%d] inventoryType [%d]",
			e.CharacterId, e.AccountId, e.Body.AssetId, e.Body.Slot, inventoryType)

		// Try to fetch projection data first, fall back to storage data
		storageProc := storage.NewProcessor(l, ctx)
		projData, err := storageProc.GetProjectionData(e.CharacterId)
		if err != nil {
			l.WithError(err).Debugf("Projection not found for character [%d], falling back to storage data", e.CharacterId)
			// Fallback to storage data
			storageData, err := storageProc.GetStorageData(e.AccountId, e.WorldId)
			if err != nil {
				l.WithError(err).Errorf("Unable to fetch storage data for account [%d] after ACCEPTED event.", e.AccountId)
				return
			}

			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
				session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
					writer.StorageOperationUpdateAssetsForCompartmentBody(l, sc.Tenant())(writer.StorageOperationModeStoreAssets, storageData.Capacity, inventoryType, storageData.Assets)))
			if err != nil {
				l.WithError(err).Errorf("Unable to send storage update to character [%d].", e.CharacterId)
			}
			return
		}

		// Get assets from the specific compartment in the projection
		compartmentName := inventoryTypeName(inventoryType)
		assets := projData.Compartments[compartmentName]

		// Send updated storage assets for the affected compartment only
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationUpdateAssetsForCompartmentBody(l, sc.Tenant())(writer.StorageOperationModeStoreAssets, projData.Capacity, inventoryType, assets)))
		if err != nil {
			l.WithError(err).Errorf("Unable to send storage update to character [%d].", e.CharacterId)
		}
	}
}

func handleStorageCompartmentReleasedEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StorageCompartmentEvent[storage2.CompartmentReleasedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StorageCompartmentEvent[storage2.CompartmentReleasedEventBody]) {
		if e.Type != storage2.StatusEventTypeCompartmentReleased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.WorldId != sc.WorldId() {
			return
		}

		if e.CharacterId == 0 {
			l.Warnf("Received storage compartment RELEASED event for account [%d] but no character ID specified", e.AccountId)
			return
		}

		inventoryType := asset.InventoryType(e.Body.InventoryType)
		l.Debugf("Storage compartment RELEASED: character [%d] account [%d] asset [%d] inventoryType [%d]",
			e.CharacterId, e.AccountId, e.Body.AssetId, inventoryType)

		// Try to fetch projection data first, fall back to storage data
		storageProc := storage.NewProcessor(l, ctx)
		projData, err := storageProc.GetProjectionData(e.CharacterId)
		if err != nil {
			l.WithError(err).Debugf("Projection not found for character [%d], falling back to storage data", e.CharacterId)
			// Fallback to storage data
			storageData, err := storageProc.GetStorageData(e.AccountId, e.WorldId)
			if err != nil {
				l.WithError(err).Errorf("Unable to fetch storage data for account [%d] after RELEASED event.", e.AccountId)
				return
			}

			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
				session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
					writer.StorageOperationUpdateAssetsForCompartmentBody(l, sc.Tenant())(writer.StorageOperationModeRetrieveAssets, storageData.Capacity, inventoryType, storageData.Assets)))
			if err != nil {
				l.WithError(err).Errorf("Unable to send storage update to character [%d].", e.CharacterId)
			}
			return
		}

		// Get assets from the specific compartment in the projection
		compartmentName := inventoryTypeName(inventoryType)
		assets := projData.Compartments[compartmentName]

		// Send updated storage assets for the affected compartment only
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
			session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationUpdateAssetsForCompartmentBody(l, sc.Tenant())(writer.StorageOperationModeRetrieveAssets, projData.Capacity, inventoryType, assets)))
		if err != nil {
			l.WithError(err).Errorf("Unable to send storage update to character [%d].", e.CharacterId)
		}
	}
}

func handleProjectionCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[storage2.StatusEvent[storage2.ProjectionCreatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e storage2.StatusEvent[storage2.ProjectionCreatedEventBody]) {
		if e.Type != storage2.StatusEventTypeProjectionCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if e.Body.WorldId != sc.WorldId() {
			return
		}

		if !sc.Is(t, e.Body.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Received PROJECTION_CREATED event for character [%d], account [%d], NPC [%d]",
			e.Body.CharacterId, e.Body.AccountId, e.Body.NpcId)

		// Fetch projection data from storage service
		storageProc := storage.NewProcessor(l, ctx)
		projData, err := storageProc.GetProjectionData(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to fetch projection data for character [%d]", e.Body.CharacterId)
			return
		}

		// Send storage UI to the character and set the storage NPC ID on the session
		sessionProc := session.NewProcessor(l, ctx)
		err = sessionProc.IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, func(s session.Model) error {
			// Store the NPC ID for fee lookup during storage operations
			sessionProc.SetStorageNpcId(s.SessionId(), e.Body.NpcId)

			// Get all assets from the projection (use equip compartment which has all initially)
			assets := projData.GetAllAssetsFromProjection()
			sort.Slice(assets, func(i, j int) bool {
				return assets[i].InventoryType() < assets[j].InventoryType()
			})

			return session.Announce(l)(ctx)(wp)(writer.StorageOperation)(
				writer.StorageOperationShowBody(l, sc.Tenant())(e.Body.NpcId, projData.Capacity, projData.Mesos, assets))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to show storage to character [%d].", e.Body.CharacterId)
		}
	}
}

func inventoryTypeName(t asset.InventoryType) string {
	switch t {
	case asset.InventoryTypeEquip:
		return "equip"
	case asset.InventoryTypeUse:
		return "use"
	case asset.InventoryTypeSetup:
		return "setup"
	case asset.InventoryTypeEtc:
		return "etc"
	case asset.InventoryTypeCash:
		return "cash"
	default:
		return "unknown"
	}
}
