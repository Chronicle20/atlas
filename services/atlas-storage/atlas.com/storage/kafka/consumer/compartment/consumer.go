package compartment

import (
	"atlas-storage/asset"
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message/compartment"
	"atlas-storage/projection"
	"atlas-storage/storage"
	"context"

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
			rf(consumer2.NewConfig(l)("storage_compartment_command")(compartment.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(compartment.EnvCommandTopic)()
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleAcceptCommand(db))))
			_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleReleaseCommand(db))))
		}
	}
}

func handleAcceptCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.AcceptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.AcceptCommandBody]) {
		if c.Type != compartment.CommandAccept {
			return
		}

		err := storage.NewProcessor(l, ctx, db).AcceptAndEmit(c.WorldId, c.AccountId, c.CharacterId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to accept item for account [%d] world [%d] transaction [%s].", c.AccountId, c.WorldId, c.Body.TransactionId)
			return
		}

		// Update projection if it exists
		inventoryType := asset.InventoryTypeFromTemplateId(c.Body.TemplateId)
		updateProjectionOnAccept(l, ctx, db, c.CharacterId, inventoryType)
	}
}

func handleReleaseCommand(db *gorm.DB) kafkaMessage.Handler[compartment.Command[compartment.ReleaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment.Command[compartment.ReleaseCommandBody]) {
		if c.Type != compartment.CommandRelease {
			return
		}

		// Get the asset before release to know its inventory type
		t := tenant.MustFromContext(ctx)
		assetModel, err := asset.GetById(db, t.Id())(uint32(c.Body.AssetId))
		if err != nil {
			l.WithError(err).Errorf("Unable to get asset [%d] before release", c.Body.AssetId)
			return
		}
		inventoryType := assetModel.InventoryType()

		err = storage.NewProcessor(l, ctx, db).ReleaseAndEmit(c.WorldId, c.AccountId, c.CharacterId, c.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to release asset [%d] for account [%d] world [%d] transaction [%s].", c.Body.AssetId, c.AccountId, c.WorldId, c.Body.TransactionId)
			return
		}

		// Update projection if it exists
		updateProjectionOnRelease(l, ctx, db, c.CharacterId, inventoryType)
	}
}

// updateProjectionOnAccept updates the projection when an asset is accepted into storage
func updateProjectionOnAccept(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, characterId uint32, inventoryType asset.InventoryType) {
	proj, ok := projection.GetManager().Get(characterId)
	if !ok {
		return // No projection exists, nothing to update
	}

	t := tenant.MustFromContext(ctx)

	// Get fresh assets from database for this storage
	assets, err := asset.GetByStorageId(db, t.Id())(proj.StorageId())
	if err != nil {
		l.WithError(err).Warnf("Failed to refresh assets for projection update")
		return
	}

	// Decorate the assets
	assetProcessor := asset.NewProcessor(l, ctx, db)
	decoratedAssets, err := assetProcessor.DecorateAll(assets)
	if err != nil {
		l.WithError(err).Warnf("Failed to decorate assets for projection update")
		decoratedAssets = assets
	}

	// Update only the operated compartment with filtered assets
	projection.GetManager().Update(characterId, func(p projection.Model) projection.Model {
		newCompartments := p.Compartments()

		// Filter the operated compartment to only matching inventory types
		filtered := make([]asset.Model[any], 0)
		for _, a := range decoratedAssets {
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
func updateProjectionOnRelease(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, characterId uint32, inventoryType asset.InventoryType) {
	proj, ok := projection.GetManager().Get(characterId)
	if !ok {
		return // No projection exists, nothing to update
	}

	t := tenant.MustFromContext(ctx)

	// Get fresh assets from database for this storage
	assets, err := asset.GetByStorageId(db, t.Id())(proj.StorageId())
	if err != nil {
		l.WithError(err).Warnf("Failed to refresh assets for projection update")
		return
	}

	// Decorate the assets
	assetProcessor := asset.NewProcessor(l, ctx, db)
	decoratedAssets, err := assetProcessor.DecorateAll(assets)
	if err != nil {
		l.WithError(err).Warnf("Failed to decorate assets for projection update")
		decoratedAssets = assets
	}

	// Update only the operated compartment with filtered assets
	projection.GetManager().Update(characterId, func(p projection.Model) projection.Model {
		newCompartments := p.Compartments()

		// Filter the operated compartment to only matching inventory types
		filtered := make([]asset.Model[any], 0)
		for _, a := range decoratedAssets {
			if a.InventoryType() == inventoryType {
				filtered = append(filtered, a)
			}
		}
		newCompartments[inventoryType] = filtered

		return projection.Clone(p).SetCompartments(newCompartments).MustBuild()
	})

	l.Debugf("Updated projection for character [%d] after RELEASE in compartment [%d]", characterId, inventoryType)
}
