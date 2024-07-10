package character

import (
	"atlas-character-factory/kafka"
	"atlas-character-factory/tenant"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"sync"
)

const (
	consumerCharacterCreated = "character_created"
	consumerItemGained       = "character_gained_item"
)

func CreatedConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerCharacterCreated)(EnvEventTopicCharacterCreated)(groupId)
	}
}

func createdValidator(tenant tenant.Model, name string) func(event createdEvent) bool {
	return func(event createdEvent) bool {
		if !tenant.Equals(event.Tenant) {
			return false
		}
		if name != event.Name {
			return false
		}
		return true
	}
}

func createdHandler(wg *sync.WaitGroup) func(idOperator model.Operator[uint32]) message.Handler[createdEvent] {
	return func(idOperator model.Operator[uint32]) message.Handler[createdEvent] {
		return func(l logrus.FieldLogger, span opentracing.Span, m createdEvent) {
			_ = idOperator(m.CharacterId)
			wg.Done()
		}
	}
}

func AwaitCreated(l logrus.FieldLogger, tenant tenant.Model) func(wg *sync.WaitGroup) func(name string, idOperator model.Operator[uint32]) {
	t := kafka.LookupTopic(l)(EnvEventTopicCharacterCreated)
	return func(wg *sync.WaitGroup) func(name string, idOperator model.Operator[uint32]) {
		return func(name string, idOperator model.Operator[uint32]) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(createdValidator(tenant, name), createdHandler(wg)(idOperator))))
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

func itemGainedHandler(wg *sync.WaitGroup) func(itemIdOperator model.Operator[uint32]) message.Handler[gainItemEvent] {
	return func(itemIdOperator model.Operator[uint32]) message.Handler[gainItemEvent] {
		return func(l logrus.FieldLogger, span opentracing.Span, m gainItemEvent) {
			_ = itemIdOperator(m.ItemId)
			wg.Done()
		}
	}
}

func AwaitItemGained(l logrus.FieldLogger, tenant tenant.Model) func(wg *sync.WaitGroup) func(itemId uint32, itemIdOperator model.Operator[uint32]) {
	t := kafka.LookupTopic(l)(EnvEventTopicItemGain)
	return func(wg *sync.WaitGroup) func(itemId uint32, itemIdOperator model.Operator[uint32]) {
		return func(itemId uint32, itemIdOperator model.Operator[uint32]) {
			_, _ = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(itemGainedValidator(tenant, itemId), itemGainedHandler(wg)(itemIdOperator))))
		}
	}
}
