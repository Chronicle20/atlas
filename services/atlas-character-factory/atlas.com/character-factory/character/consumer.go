package character

import (
	"atlas-character-factory/async"
	"atlas-character-factory/kafka"
	"atlas-character-factory/tenant"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	consumerCharacterCreated = "character_created"
	consumerItemGained       = "character_gained_item"
	consumerEquipChanged     = "character_equip_changed"
)

func CreatedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerCharacterCreated)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func createdValidator(tenant tenant.Model, name string) func(event statusEvent) bool {
	return func(event statusEvent) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if event.Type != EventCharacterStatusTypeCreated {
			return false
		}
		if name != event.Name {
			return false
		}
		return true
	}
}

func createdHandler(rchan chan uint32, _ chan error) message.Handler[statusEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, m statusEvent) {
		rchan <- m.CharacterId
	}
}

func AwaitCreated(l logrus.FieldLogger, tenant tenant.Model) func(name string) async.Provider[uint32] {
	t := kafka.LookupTopic(l)(EnvEventTopicCharacterStatus)
	return func(name string) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			l.Debugf("Creating OneTime topic consumer to await [%s] character creation.", name)
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant, name), createdHandler(rchan, echan))))
		}
	}
}

func ItemGainedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerItemGained)(EnvEventTopicItemGain)(groupId)
	}
}

func itemGainedValidator(tenant tenant.Model, itemId uint32) func(event gainItemEvent) bool {
	return func(event gainItemEvent) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if itemId != event.ItemId {
			return false
		}
		return true
	}
}

func itemGainedHandler(rchan chan ItemGained, _ chan error) message.Handler[gainItemEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, m gainItemEvent) {
		rchan <- ItemGained{ItemId: m.ItemId, Slot: m.Slot}
	}
}

func AwaitItemGained(l logrus.FieldLogger, tenant tenant.Model) func(itemId uint32) async.Provider[ItemGained] {
	t := kafka.LookupTopic(l)(EnvEventTopicItemGain)
	return func(itemId uint32) async.Provider[ItemGained] {
		return func(ctx context.Context, rchan chan ItemGained, echan chan error) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(itemGainedValidator(tenant, itemId), itemGainedHandler(rchan, echan))))
		}
	}
}

func EquipChangedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerEquipChanged)(EnvEventTopicEquipChanged)(groupId)
	}
}

func equipChangedValidator(tenant tenant.Model, itemId uint32) func(event equipChangedEvent) bool {
	return func(event equipChangedEvent) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if itemId != event.ItemId {
			return false
		}
		if "EQUIPPED" != event.Change {
			return false
		}
		return true
	}
}

func equipChangedHandler(rchan chan uint32, _ chan error) message.Handler[equipChangedEvent] {
	return func(l logrus.FieldLogger, span opentracing.Span, m equipChangedEvent) {
		rchan <- m.ItemId
	}
}

func AwaitEquipChanged(l logrus.FieldLogger, tenant tenant.Model) func(itemId uint32) async.Provider[uint32] {
	t := kafka.LookupTopic(l)(EnvEventTopicEquipChanged)
	return func(itemId uint32) async.Provider[uint32] {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(equipChangedValidator(tenant, itemId), equipChangedHandler(rchan, echan))))
		}
	}
}
