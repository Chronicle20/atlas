package asset

import (
	"atlas-character-factory/asset"
	"atlas-character-factory/compartment"
	consumer2 "atlas-character-factory/kafka/consumer"
	asset2 "atlas-character-factory/kafka/message/asset"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func createdHandler(rchan chan asset.Model, _ chan error) message.Handler[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
		l.Debugf("Asset [%d] created for character [%d].", e.AssetId, e.CharacterId)
		rchan <- asset.NewModel(e.AssetId, e.TemplateId, e.Slot)
	}
}

func createdValidator(t tenant.Model) func(characterId uint32, compartmentId uuid.UUID, templateId uint32) message.Validator[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
	return func(characterId uint32, compartmentId uuid.UUID, templateId uint32) message.Validator[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) bool {
			if e.Type != asset2.StatusEventTypeCreated {
				return false
			}
			if !t.Is(tenant.MustFromContext(ctx)) {
				return false
			}
			if characterId != e.CharacterId {
				return false
			}
			if compartmentId != e.CompartmentId {
				return false
			}
			if templateId != e.TemplateId {
				return false
			}
			return true
		}
	}
}

func AwaitCreated(l logrus.FieldLogger) func(characterId uint32, compartmentId uuid.UUID, templateId uint32, inventoryType inventory.Type) async.Provider[asset.Model] {
	t, _ := topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
	return func(characterId uint32, compartmentId uuid.UUID, templateId uint32, inventoryType inventory.Type) async.Provider[asset.Model] {
		return func(ctx context.Context, rchan chan asset.Model, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await asset of template [%d] creation in compartment [%s] for character [%d].", templateId, compartmentId.String(), characterId)
			hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant.MustFromContext(ctx))(characterId, compartmentId, templateId), createdHandler(rchan, echan))))
			if err != nil {
				echan <- err
			}
			l.Debugf("Generating asset [%d] and for character.", templateId)
			err = compartment.NewProcessor(l, ctx).CreateAsset(characterId, inventoryType, templateId, 1, time.Time{}, characterId)
			if err != nil {
				echan <- err
			}
			go func() {
				<-ctx.Done()
				l.Debugf("Removing handler [%s].", hid)
				_ = consumer.GetManager().RemoveHandler(t, hid)
			}()
		}
	}
}

func slotUpdateHandler(rchan chan uint32, echan chan error) message.Handler[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
		l.Debugf("Asset [%d] moved for character [%d].", e.AssetId, e.CharacterId)
		rchan <- e.AssetId
	}
}

func slotUpdateValidator(t tenant.Model) func(characterId uint32, compartmentId uuid.UUID, assetId uint32) message.Validator[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
	return func(characterId uint32, compartmentId uuid.UUID, assetId uint32) message.Validator[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) bool {
			if e.Type != asset2.StatusEventTypeMoved {
				return false
			}
			if !t.Is(tenant.MustFromContext(ctx)) {
				return false
			}
			if characterId != e.CharacterId {
				return false
			}
			if compartmentId != e.CompartmentId {
				return false
			}
			if assetId != e.AssetId {
				return false
			}
			return true
		}
	}
}

func AwaitSlotUpdate(l logrus.FieldLogger) func(characterId uint32, compartmentId uuid.UUID, assetId uint32) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
	return func(characterId uint32, compartmentId uuid.UUID, assetId uint32) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await asset [%d] slot update in compartment [%s] for character [%d].", assetId, compartmentId.String(), characterId)
			hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(slotUpdateValidator(tenant.MustFromContext(ctx))(characterId, compartmentId, assetId), slotUpdateHandler(rchan, echan))))
			if err != nil {
				echan <- err
			}
			go func() {
				<-ctx.Done()
				l.Debugf("Removing handler [%s].", hid)
				_ = consumer.GetManager().RemoveHandler(t, hid)
			}()
		}
	}
}
