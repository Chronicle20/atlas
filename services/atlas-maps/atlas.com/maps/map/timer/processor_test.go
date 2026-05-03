package timer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// recordingProducer captures emitted messages by topic — copied from
// services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go.
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

func mkProcTenant(t *testing.T) tenant.Model {
	t.Helper()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tt
}

func newTestProcessor(t *testing.T, reg *Registry, rec *recordingProducer, tt tenant.Model) Processor {
	t.Helper()
	logger, _ := test.NewNullLogger()
	tctx := tenant.WithContext(context.Background(), tt)
	return NewProcessorWithRegistry(logger, tctx, rec.Provider(), reg)
}

func TestProcessor_Register_InsertsEntryAndEmitsMapTimerStarted(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	got, ok := reg.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, _map.Id(100000201), got.ForcedReturnMapId())
	require.Equal(t, uint32(600), got.Seconds())

	msgs := rec.Messages(mapKafka.EnvEventTopicMapStatus)
	require.Len(t, msgs, 1)
	var ev mapKafka.StatusEvent[mapKafka.MapTimerStarted]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, mapKafka.EventTopicMapStatusTypeMapTimerStarted, ev.Type)
	require.Equal(t, uint32(42), ev.Body.CharacterId)
	require.Equal(t, uint32(600), ev.Body.Seconds)
}

func TestProcessor_Register_ReplacesPriorEntry(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f1 := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f2 := field.NewBuilder(0, 0, _map.Id(200000000)).SetInstance(uuid.Nil).Build()

	require.NoError(t, p.Register(uuid.New(), uint32(42), f1, _map.Id(100000201), uint32(600)))
	first, ok := reg.Get(tt, 42)
	require.True(t, ok)

	require.NoError(t, p.Register(uuid.New(), uint32(42), f2, _map.Id(200000201), uint32(300)))
	second, ok := reg.Get(tt, 42)
	require.True(t, ok)
	require.NotEqual(t, first.Token(), second.Token(), "second Register must mint a new token")
	require.Equal(t, _map.Id(200000201), second.ForcedReturnMapId(), "second Register replaces forcedReturnMapId")
}

func TestProcessor_CancelIfTracked_RemovesAndStops(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	cancelled := p.CancelIfTracked(uint32(42))
	require.True(t, cancelled)

	_, ok := reg.Get(tt, 42)
	require.False(t, ok, "CancelIfTracked must remove entry")
}

func TestProcessor_CancelIfTracked_AbsentReturnsFalse(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)
	require.False(t, p.CancelIfTracked(uint32(999)))
}
