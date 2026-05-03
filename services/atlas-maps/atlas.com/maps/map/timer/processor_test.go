package timer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	characterKafka "atlas-maps/kafka/message/character"
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

func TestProcessor_ForceReturnIfTracked_EmitsChangeMap(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	forced := p.ForceReturnIfTracked(uint32(42))
	require.True(t, forced)

	_, ok := reg.Get(tt, 42)
	require.False(t, ok)

	msgs := rec.Messages(characterKafka.EnvCommandTopic)
	require.Len(t, msgs, 1)
	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, characterKafka.CommandChangeMap, cmd.Type)
	require.Equal(t, world.Id(1), cmd.WorldId)
	require.Equal(t, channel.Id(2), cmd.Body.ChannelId)
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
	require.Equal(t, uuid.Nil, cmd.Body.Instance)
	require.Equal(t, uint32(0), cmd.Body.PortalId)
}

func TestProcessor_ForceReturnIfTracked_AbsentIsNoOp(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)
	require.False(t, p.ForceReturnIfTracked(uint32(999)))
	require.Empty(t, rec.Messages(characterKafka.EnvCommandTopic))
}

func TestProcessor_TimerFires_EmitsChangeMap(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(0)))

	time.Sleep(150 * time.Millisecond)

	_, ok := reg.Get(tt, 42)
	require.False(t, ok, "expired entry must be removed by handleExpire")

	msgs := rec.Messages(characterKafka.EnvCommandTopic)
	require.Len(t, msgs, 1)
	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
}

func TestProcessor_TimerFires_StaleTokenNoOp(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(60)))

	// Capture token for the first entry, then directly invoke handleExpire
	// with the captured (stale) token after replacement. This deterministically
	// simulates a 0-second timer goroutine firing AFTER replacement, which the
	// AfterFunc(0) variant cannot guarantee due to scheduling races.
	first, ok := reg.Get(tt, 42)
	require.True(t, ok)
	staleToken := first.Token()

	f2 := field.NewBuilder(0, 0, _map.Id(200000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f2, _map.Id(200000201), uint32(60)))

	impl := p.(*ProcessorImpl)
	impl.handleExpire(tt, 42, staleToken)

	got, ok := reg.Get(tt, 42)
	require.True(t, ok, "second entry must still be present")
	require.Equal(t, _map.Id(200000201), got.ForcedReturnMapId(), "second entry survived")
	require.Empty(t, rec.Messages(characterKafka.EnvCommandTopic), "stale token must not emit CHANGE_MAP")
}
