package mist

import (
	"atlas-maps/kafka/producer"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// recordingProducer captures emitted messages by topic for assertions
// without going through Kafka.
type recordingProducer struct {
	mu       sync.Mutex
	messages map[string][]kafka.Message
}

func newRecordingProducer() *recordingProducer {
	return &recordingProducer{messages: map[string][]kafka.Message{}}
}

func (m *recordingProducer) Provider() producer.Provider {
	return func(token string) kafkaProducer.MessageProducer {
		return func(prov model.Provider[[]kafka.Message]) error {
			msgs, err := prov()
			if err != nil {
				return err
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages[token] = append(m.messages[token], msgs...)
			return nil
		}
	}
}

func (m *recordingProducer) Messages(topic string) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]kafka.Message(nil), m.messages[topic]...)
}

func newTestMistProcessor(t *testing.T, tt tenant.Model, rec *recordingProducer) (*ProcessorImpl, context.Context) {
	t.Helper()
	logger, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)
	return &ProcessorImpl{
		l:   logger,
		ctx: ctx,
		t:   tt,
		p:   rec.Provider(),
		r:   newTestMistRegistry(),
	}, ctx
}

func TestProcessor_Create_AddsToRegistryAndEmitsCreated(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "MONSTER", OwnerId: 9001,
		OriginX: 100, OriginY: 200,
		LtX: -50, LtY: -30, RbX: 50, RbY: 30,
		Disease: "POISON", DiseaseValue: 80, DiseaseDuration: 30000,
		Duration: 10000, TickIntervalMs: 1000,
		SourceSkillId: 100020, SourceSkillLevel: 5,
	}

	m, err := p.Create(body)
	require.NoError(t, err)
	require.Equal(t, "POISON", m.Disease())
	require.Equal(t, uint32(9001), m.OwnerId())
	require.Equal(t, "MONSTER", m.OwnerType())

	// Registry side: mist is present under tenant.
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	got := p.r.GetByField(tt, f)
	require.Len(t, got, 1)
	require.Equal(t, m.Id(), got[0].Id())

	// Producer side: a single MIST_CREATED event was emitted.
	msgs := rec.Messages(mistKafka.EnvEventTopic)
	require.Len(t, msgs, 1, "expected exactly one MIST_CREATED message")

	var event mistKafka.Event[mistKafka.CreatedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeCreated, event.Type)
	require.Equal(t, m.Id(), event.MistId)
	require.Equal(t, tt.Id(), event.Tenant)
	require.Equal(t, int64(10000), event.Body.Duration)
	require.Equal(t, "MONSTER", event.Body.OwnerType)
	require.Equal(t, uint32(9001), event.Body.OwnerId)
	require.Equal(t, int16(100), event.Body.OriginX)
	require.Equal(t, int16(-50), event.Body.LtX)
	require.Equal(t, int16(50), event.Body.RbX)
}

func TestProcessor_Destroy_RemovesAndEmitsDestroyed(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	body := mistKafka.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		Disease: "POISON", Duration: 10000, TickIntervalMs: 1000,
	}
	m, err := p.Create(body)
	require.NoError(t, err)

	removed, err := p.Destroy(m.Id(), mistKafka.ReasonExpired)
	require.NoError(t, err)
	require.Equal(t, m.Id(), removed.Id())

	// Registry side: mist is gone.
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	require.Empty(t, p.r.GetByField(tt, f))

	// Producer side: one MIST_CREATED followed by one MIST_DESTROYED.
	msgs := rec.Messages(mistKafka.EnvEventTopic)
	require.Len(t, msgs, 2)

	var destroyed mistKafka.Event[mistKafka.DestroyedBody]
	require.NoError(t, json.Unmarshal(msgs[1].Value, &destroyed))
	require.Equal(t, mistKafka.EventTypeDestroyed, destroyed.Type)
	require.Equal(t, mistKafka.ReasonExpired, destroyed.Body.Reason)
	require.Equal(t, m.Id(), destroyed.MistId)
}

func TestProcessor_Destroy_NotFound_ReturnsError(t *testing.T) {
	tt := mkRegTenant()
	rec := newRecordingProducer()
	p, _ := newTestMistProcessor(t, tt, rec)

	_, err := p.Destroy(uuid.New(), mistKafka.ReasonCancelled)
	require.Error(t, err)
	require.Empty(t, rec.Messages(mistKafka.EnvEventTopic))
}

func TestCreatedEventProvider_BuildsCreatedEvent(t *testing.T) {
	tt := mkRegTenant()
	id := uuid.New()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	m := NewBuilder(id, f).
		SetOwner("MONSTER", 9001).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(10 * time.Second).
		Build()

	prov := createdEventProvider(tt, m)
	require.NotNil(t, prov)
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var event mistKafka.Event[mistKafka.CreatedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeCreated, event.Type)
	require.Equal(t, id, event.MistId)
	require.Equal(t, tt.Id(), event.Tenant)
	require.Equal(t, int16(100), event.Body.OriginX)
	require.Equal(t, int64(10000), event.Body.Duration)
}

func TestDestroyedEventProvider_BuildsDestroyedEvent(t *testing.T) {
	tt := mkRegTenant()
	id := uuid.New()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	m := NewBuilder(id, f).Build()
	prov := destroyedEventProvider(tt, m, mistKafka.ReasonExpired)
	require.NotNil(t, prov)
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var event mistKafka.Event[mistKafka.DestroyedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeDestroyed, event.Type)
	require.Equal(t, mistKafka.ReasonExpired, event.Body.Reason)
	require.Equal(t, id, event.MistId)
}
