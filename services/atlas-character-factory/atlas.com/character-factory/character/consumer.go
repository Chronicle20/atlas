package character

import (
	consumer2 "atlas-character-factory/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/async"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func createdValidator(t tenant.Model) func(name string) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventCreatedBody]) bool {
	return func(name string) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventCreatedBody]) bool {
		return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventCreatedBody]) bool {
			if !t.Is(tenant.MustFromContext(ctx)) {
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
}

func createdHandler(rchan chan uint32, _ chan error) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, m statusEvent[statusEventCreatedBody]) {
		rchan <- m.CharacterId
	}
}

func AwaitCreated(l logrus.FieldLogger) func(name string) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
	return func(name string) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await [%s] character creation.", name)
			_, err := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant.MustFromContext(ctx))(name), createdHandler(rchan, echan))))
			if err != nil {
				echan <- err
			}
		}
	}
}

func ItemGainedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerInventoryChanged)(EnvEventInventoryChanged)(groupId)
	}
}

func itemGainedValidator(t tenant.Model) func(characterId uint32) func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
	return func(characterId uint32) func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
		return func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
			return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemAddBody]) bool {
				if !t.Is(tenant.MustFromContext(ctx)) {
					return false
				}
				if characterId != event.CharacterId {
					return false
				}
				if itemId != event.Body.ItemId {
					return false
				}
				return true
			}
		}
	}
}

func itemGainedHandler(rchan chan ItemGained, _ chan error) message.Handler[inventoryChangedEvent[inventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, m inventoryChangedEvent[inventoryChangedItemAddBody]) {
		rchan <- ItemGained{ItemId: m.Body.ItemId, Slot: m.Slot}
	}
}

func AwaitItemGained(l logrus.FieldLogger) func(characterId uint32) func(itemId uint32) async.Provider[ItemGained] {
	t, _ := topic.EnvProvider(l)(EnvEventInventoryChanged)()
	return func(characterId uint32) func(itemId uint32) async.Provider[ItemGained] {
		return func(itemId uint32) async.Provider[ItemGained] {
			return func(ctx context.Context, rchan chan ItemGained, echan chan error) {
				tenant := tenant.MustFromContext(ctx)
				_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(itemGainedValidator(tenant)(characterId)(itemId), itemGainedHandler(rchan, echan))))
			}
		}
	}
}

func EquipChangedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerInventoryChanged)(EnvEventInventoryChanged)(groupId)
	}
}

func equipChangedValidator(t tenant.Model) func(characterId uint32) func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
	return func(characterId uint32) func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
		return func(itemId uint32) func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
			return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) bool {
				if !t.Is(tenant.MustFromContext(ctx)) {
					return false
				}
				if characterId != event.CharacterId {
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
	}
}

func equipChangedHandler(rchan chan uint32, _ chan error) message.Handler[inventoryChangedEvent[inventoryChangedItemMoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, m inventoryChangedEvent[inventoryChangedItemMoveBody]) {
		rchan <- m.Body.ItemId
	}
}

func AwaitEquipChanged(l logrus.FieldLogger) func(characterId uint32) func(itemId uint32) async.Provider[uint32] {
	t, _ := topic.EnvProvider(l)(EnvEventInventoryChanged)()
	return func(characterId uint32) func(itemId uint32) async.Provider[uint32] {
		return func(itemId uint32) async.Provider[uint32] {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				tenant := tenant.MustFromContext(ctx)
				_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(equipChangedValidator(tenant)(characterId)(itemId), equipChangedHandler(rchan, echan))))
			}
		}
	}
}
