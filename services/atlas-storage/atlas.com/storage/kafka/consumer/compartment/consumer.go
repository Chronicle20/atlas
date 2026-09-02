package compartment

import (
	"atlas-storage/asset"
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message/compartment"
	"atlas-storage/projection"
	"atlas-storage/storage"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("storage_compartment_command")(compartment.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(compartment.EnvCommandTopic)()
			if err != nil {
				return err
			}
			if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleAcceptCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleReleaseCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleAcceptCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.AcceptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.AcceptCommandBody]) {
		if c.Type != compartment.CommandAccept {
			return
		}

		// Guarded: ACCEPT creates a durable storage asset and Kafka delivery is
		// at-least-once (task-208). The claim lives in the processor so it can
		// cover the DB write without enclosing the direct Kafka emit.
		err := storage.NewProcessor(l, ctx, db).AcceptOnceAndEmit(c.WorldId, c.AccountId, c.CharacterId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to accept item for account [%d] world [%d] transaction [%s].", c.AccountId, c.WorldId, c.Body.TransactionId)
			return
		}

		// Update projection if it exists
		inventoryType, _ := inventory.TypeFromItemId(item.Id(c.Body.TemplateId))
		updateProjectionOnAccept(l, ctx, db, c.CharacterId, inventoryType)
	}
}

func handleReleaseCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.ReleaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.ReleaseCommandBody]) {
		if c.Type != compartment.CommandRelease {
			return
		}

		// Read the asset up front only to learn its inventory type for the
		// projection refresh below. A missing row means the release already
		// happened — the ordinary shape of an at-least-once redelivery, not an
		// error (task-208).
		assetModel, err := asset.GetById(db.WithContext(ctx))(uint32(c.Body.AssetId))
		if err != nil {
			l.Infof("Asset [%d] is already gone for transaction [%s]; treating RELEASE as already applied.", c.Body.AssetId, c.Body.TransactionId)
			return
		}
		inventoryType := assetModel.InventoryType()

		// Guarded: a redelivered RELEASE must not release the asset twice
		// (task-208). Claim lives in the processor — see AcceptOnceAndEmit.
		err = storage.NewProcessor(l, ctx, db).ReleaseOnceAndEmit(c.WorldId, c.AccountId, c.CharacterId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to release asset [%d] for account [%d] world [%d] transaction [%s].", c.Body.AssetId, c.AccountId, c.WorldId, c.Body.TransactionId)
			return
		}

		// Update projection if it exists
		updateProjectionOnRelease(l, ctx, db, c.CharacterId, inventoryType)
	}
}

// updateProjectionOnAccept updates the projection when an asset is accepted into storage
func updateProjectionOnAccept(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, characterId uint32, inventoryType inventory.Type) {
	proj, ok := projection.GetManager().Get(ctx, characterId)
	if !ok {
		return // No projection exists, nothing to update
	}

	// Get fresh assets from database for this storage
	assets, err := asset.GetByStorageId(db.WithContext(ctx))(proj.StorageId())
	if err != nil {
		l.WithError(err).Warnf("Failed to refresh assets for projection update")
		return
	}

	// Update only the operated compartment with filtered assets
	projection.GetManager().Update(ctx, characterId, func(p projection.Model) projection.Model {
		newCompartments := p.Compartments()

		filtered := make([]asset.Model, 0)
		for _, a := range assets {
			if a.InventoryType() == inventoryType {
				filtered = append(filtered, a)
			}
		}
		newCompartments[inventoryType] = filtered

		return projection.Clone(p).SetCompartments(newCompartments).MustBuild()
	})

	l.Debugf("Updated projection for character [%d] after ACCEPT in compartment [%d]", characterId, inventoryType)
}

// updateProjectionOnRelease updates the projection when an asset is released from storage
func updateProjectionOnRelease(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, characterId uint32, inventoryType inventory.Type) {
	proj, ok := projection.GetManager().Get(ctx, characterId)
	if !ok {
		return // No projection exists, nothing to update
	}

	// Get fresh assets from database for this storage
	assets, err := asset.GetByStorageId(db.WithContext(ctx))(proj.StorageId())
	if err != nil {
		l.WithError(err).Warnf("Failed to refresh assets for projection update")
		return
	}

	// Update only the operated compartment with filtered assets
	projection.GetManager().Update(ctx, characterId, func(p projection.Model) projection.Model {
		newCompartments := p.Compartments()

		filtered := make([]asset.Model, 0)
		for _, a := range assets {
			if a.InventoryType() == inventoryType {
				filtered = append(filtered, a)
			}
		}
		newCompartments[inventoryType] = filtered

		return projection.Clone(p).SetCompartments(newCompartments).MustBuild()
	})

	l.Debugf("Updated projection for character [%d] after RELEASE in compartment [%d]", characterId, inventoryType)
}
