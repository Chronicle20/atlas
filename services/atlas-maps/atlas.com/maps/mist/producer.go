package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
)

// createdEventProvider builds a MIST_CREATED event keyed by the mist id.
func createdEventProvider(t tenant.Model, m Mist) model.Provider[[]kafka.Message] {
	key := []byte(m.Id().String())
	value := &mistKafka.Event[mistKafka.CreatedBody]{
		Tenant:    t.Id(),
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MistId:    m.Id(),
		Type:      mistKafka.EventTypeCreated,
		Body: mistKafka.CreatedBody{
			OwnerType: m.OwnerType(),
			OwnerId:   m.OwnerId(),
			OriginX:   m.OriginX(),
			OriginY:   m.OriginY(),
			LtX:       m.LtX(),
			LtY:       m.LtY(),
			RbX:       m.RbX(),
			RbY:       m.RbY(),
			Duration:  int64(m.Duration() / time.Millisecond),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// destroyedEventProvider builds a MIST_DESTROYED event keyed by the mist id.
func destroyedEventProvider(t tenant.Model, m Mist, reason string) model.Provider[[]kafka.Message] {
	key := []byte(m.Id().String())
	value := &mistKafka.Event[mistKafka.DestroyedBody]{
		Tenant:    t.Id(),
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MistId:    m.Id(),
		Type:      mistKafka.EventTypeDestroyed,
		Body:      mistKafka.DestroyedBody{Reason: reason},
	}
	return producer.SingleMessageProvider(key, value)
}
