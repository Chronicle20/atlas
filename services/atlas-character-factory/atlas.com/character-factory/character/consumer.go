package character

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	"atlas-character-factory/tenant"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	consumerCharacterCreated = "character_created"
	consumerInventoryChanged = "character_inventory_changed"
)

func CreatedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerCharacterCreated)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func createdValidator(tenant tenant.Model, name string) func(event statusEvent[statusEventCreatedBody]) bool {
	return func(event statusEvent[statusEventCreatedBody]) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if event.Type != EventCharacterStatusTypeCreated {
			return false
		}
		if name != event.Body.Name {
			return false
		}
		return true
	}
}

func createdHandler(rchan chan uint32, _ chan error) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, m statusEvent[statusEventCreatedBody]) {
		rchan <- m.CharacterId
	}
}

func AwaitCreated(l logrus.FieldLogger, tenant tenant.Model) func(name string) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return func(name string) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await [%s] character creation.", name)
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant, name), createdHandler(rchan, echan))))
		}
	}
}

func ItemGainedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerInventoryChanged)(EnvEventInventoryChanged)(groupId)
	}
}

func itemGainedValidator(tenant tenant.Model, itemId uint32) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
	return func(event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if itemId != event.Body.ItemId {
			return false
		}
		return true
	}
}

func itemGainedHandler(rchan chan ItemGained, _ chan error) message.Handler[inventoryChangedEvent[inventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, m inventoryChangedEvent[inventoryChangedItemAddBody]) {
		rchan <- ItemGained{ItemId: m.Body.ItemId, Slot: m.Slot}
	}
}

func AwaitItemGained(l logrus.FieldLogger, tenant tenant.Model) func(itemId uint32) async.Provider[ItemGained] {
	t, _ := topic.EnvProvider(l)(EnvEventInventoryChanged)()
	return func(itemId uint32) async.Provider[ItemGained] {
		return func(ctx context.Context, rchan chan ItemGained, echan chan error) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(itemGainedValidator(tenant, itemId), itemGainedHandler(rchan, echan))))
		}
	}
}

func EquipChangedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerInventoryChanged)(EnvEventInventoryChanged)(groupId)
	}
}

func equipChangedValidator(tenant tenant.Model, itemId uint32) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
	return func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if itemId != event.Body.ItemId {
			return false
		}
		if event.Slot < 0 {
			return false
		}
		return true
	}
}

func equipChangedHandler(rchan chan uint32, _ chan error) message.Handler[inventoryChangedEvent[inventoryChangedItemMoveBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, m inventoryChangedEvent[inventoryChangedItemMoveBody]) {
		rchan <- m.Body.ItemId
	}
}

func AwaitEquipChanged(l logrus.FieldLogger, tenant tenant.Model) func(itemId uint32) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(EnvEventInventoryChanged)()
	return func(itemId uint32) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(equipChangedValidator(tenant, itemId), equipChangedHandler(rchan, echan))))
		}
	}
}
