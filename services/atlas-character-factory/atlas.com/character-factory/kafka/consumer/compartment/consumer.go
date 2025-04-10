package compartment

import (
	"atlas-character-factory/compartment"
	consumer2 "atlas-character-factory/kafka/consumer"
	compartment2 "atlas-character-factory/kafka/message/compartment"
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
			rf(consumer2.NewConfig(l)("compartment_status_event")(compartment2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func AwaitEquipableCreated(l logrus.FieldLogger) func(characterId uint32, equipment []uint32, equipables []uint32) async.Provider[uuid.UUID] {
	t, _ := topic.EnvProvider(l)(compartment2.EnvEventTopicStatus)()
	return func(characterId uint32, equipment []uint32, equipables []uint32) async.Provider[uuid.UUID] {
		return func(ctx context.Context, rchan chan uuid.UUID, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await compartment [%d] creation for character [%d].", inventory.TypeValueEquip, characterId)
			hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant.MustFromContext(ctx))(characterId, inventory.TypeValueEquip), equipableCreatedHandler(rchan, echan, equipment, equipables))))
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

func AwaitCreated(l logrus.FieldLogger) func(characterId uint32, inventoryType inventory.Type, items []uint32) async.Provider[uuid.UUID] {
	t, _ := topic.EnvProvider(l)(compartment2.EnvEventTopicStatus)()
	return func(characterId uint32, inventoryType inventory.Type, items []uint32) async.Provider[uuid.UUID] {
		return func(ctx context.Context, rchan chan uuid.UUID, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await compartment [%d] creation for character [%d].", inventoryType, characterId)
			hid, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant.MustFromContext(ctx))(characterId, inventoryType), otherCreatedHandler(rchan, echan, inventoryType, items))))
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

func createdValidator(t tenant.Model) func(characterId uint32, inventoryType inventory.Type) message.Validator[compartment2.StatusEvent[compartment2.CreatedStatusEventBody]] {
	return func(characterId uint32, inventoryType inventory.Type) message.Validator[compartment2.StatusEvent[compartment2.CreatedStatusEventBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, e compartment2.StatusEvent[compartment2.CreatedStatusEventBody]) bool {
			if !t.Is(tenant.MustFromContext(ctx)) {
				return false
			}
			if characterId != e.CharacterId {
				return false
			}
			if inventoryType != inventory.Type(e.Body.Type) {
				return false
			}
			return true
		}
	}
}

func equipableCreatedHandler(rchan chan uuid.UUID, echan chan error, equipment []uint32, equipables []uint32) message.Handler[compartment2.StatusEvent[compartment2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment2.StatusEvent[compartment2.CreatedStatusEventBody]) {
		l.Debugf("Compartment [%s] created for character [%d].", e.CompartmentId.String(), e.CharacterId)
		for _, aid := range equipment {
			l.Debugf("Generating equipable asset [%d] and equiping for character.", aid)
			err := compartment.NewProcessor(l, ctx).CreateAsset(e.CharacterId, inventory.TypeValueEquip, aid, 1, time.Time{}, e.CharacterId)
			if err != nil {
				echan <- err
			}
		}
		for _, aid := range equipables {
			l.Debugf("Generating equipable asset [%d] for character.", aid)
			err := compartment.NewProcessor(l, ctx).CreateAsset(e.CharacterId, inventory.TypeValueEquip, aid, 1, time.Time{}, e.CharacterId)
			if err != nil {
				echan <- err
			}
		}
		rchan <- e.CompartmentId
	}
}

func otherCreatedHandler(rchan chan uuid.UUID, echan chan error, inventoryType inventory.Type, items []uint32) message.Handler[compartment2.StatusEvent[compartment2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment2.StatusEvent[compartment2.CreatedStatusEventBody]) {
		l.Debugf("Compartment [%s] created for character [%d].", e.CompartmentId.String(), e.CharacterId)
		for _, aid := range items {
			l.Debugf("Generating asset [%d] for character.", aid)
			err := compartment.NewProcessor(l, ctx).CreateAsset(e.CharacterId, inventoryType, aid, 1, time.Time{}, e.CharacterId)
			if err != nil {
				echan <- err
			}
		}
		rchan <- e.CompartmentId
	}
}
