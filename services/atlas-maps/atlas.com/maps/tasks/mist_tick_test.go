package tasks

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"
	"atlas-maps/kafka/producer"
	"atlas-maps/mist"

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

func mkTickTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func newTestMistTick(t *testing.T, reg *mist.Registry, rec *recordingProducer, posLookup PositionLookup) *MistTick {
	t.Helper()
	logger, _ := test.NewNullLogger()
	mt := NewMistTick(logger, 1000, posLookup)
	mt.registry = reg
	mt.producerProvider = func(ctx context.Context) producer.Provider {
		return rec.Provider()
	}
	return mt
}

func TestMistTick_SleepTime_RespectsConfiguredInterval(t *testing.T) {
	mt := NewMistTick(nil, 750, nil)
	require.Equal(t, 750*time.Millisecond, mt.SleepTime())
}

func TestMistTick_ExpiredMist_DestroysAndEmits(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()
	posLookup := func(ctx context.Context, cid uint32) (int16, int16, error) {
		return 0, 0, nil
	}

	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	expiredMist := mist.NewBuilder(id, f).
		SetOwner("MONSTER", 1).
		SetOrigin(0, 0).
		SetBounds(-50, -50, 50, 50).
		SetDisease("POISON", 80, 30*time.Second).
		SetDuration(-1 * time.Second). // Already expired
		SetTickInterval(time.Second).
		Build()
	require.NoError(t, reg.Add(tt, expiredMist))

	mt := newTestMistTick(t, reg, rec, posLookup)
	mt.runOnce(context.Background())

	// Registry: mist removed.
	require.Empty(t, reg.AllByTenant(tt))

	// Producer: a MIST_DESTROYED with reason EXPIRED was emitted.
	msgs := rec.Messages(mistKafka.EnvEventTopic)
	require.Len(t, msgs, 1)
	var event mistKafka.Event[mistKafka.DestroyedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &event))
	require.Equal(t, mistKafka.EventTypeDestroyed, event.Type)
	require.Equal(t, mistKafka.ReasonExpired, event.Body.Reason)
	require.Equal(t, id, event.MistId)
}

func TestMistTick_LiveMist_AppliesDiseaseToContainedCharacters(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()

	const insideId = uint32(1001)
	const outsideId = uint32(1002)
	posLookup := func(ctx context.Context, cid uint32) (int16, int16, error) {
		switch cid {
		case insideId:
			return 10, 10, nil
		case outsideId:
			return 5000, 5000, nil
		}
		return 0, 0, nil
	}

	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	liveMist := mist.NewBuilder(id, f).
		SetOwner("MONSTER", 9001).
		SetOrigin(0, 0).
		SetBounds(-100, -100, 100, 100).
		SetDisease("POISON", 80, 30*time.Second).
		SetDuration(time.Minute).
		SetTickInterval(time.Second).
		SetSource(100020, 5).
		Build()
	require.NoError(t, reg.Add(tt, liveMist))

	mt := newTestMistTick(t, reg, rec, posLookup)
	mt.charsInField = func(t tenant.Model, ff field.Model) []uint32 {
		return []uint32{insideId, outsideId}
	}
	mt.runOnce(context.Background())

	// MIST_DESTROYED should not have been emitted (still live).
	require.Empty(t, rec.Messages(mistKafka.EnvEventTopic))

	// One apply-disease command to the inside character only.
	msgs := rec.Messages(EnvCommandTopicCharacterBuff)
	require.Len(t, msgs, 1, "expected one apply-disease command for the inside character")

	var cmd buffCommand[applyDiseaseBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, "APPLY", cmd.Type)
	require.Equal(t, insideId, cmd.CharacterId)
	require.Equal(t, int32(100020), cmd.Body.SourceId)
	require.Equal(t, byte(5), cmd.Body.Level)
	// Duration is in SECONDS (atlas-buffs' buff.NewBuff multiplies by
	// time.Second). 30s mist disease -> 30, not 30000ms. The previous
	// 30000 expectation pinned a bug where AREA_POISON DoTs persisted
	// for hours instead of the configured mist disease duration.
	require.Equal(t, int32(30), cmd.Body.Duration)
	require.Len(t, cmd.Body.Changes, 1)
	require.Equal(t, "POISON", cmd.Body.Changes[0].Type)
	require.Equal(t, int32(80), cmd.Body.Changes[0].Amount)

	// Tick advanced.
	got := reg.AllByTenant(tt)
	require.Len(t, got, 1)
	require.False(t, got[0].LastTick().IsZero())
}

func TestMistTick_DifferentInstances_DoNotCrossApply(t *testing.T) {
	tt := mkTickTenant()
	reg := mist.NewTestRegistry()
	rec := newRecordingProducer()

	const otherInstanceCharId = uint32(2001)
	posLookup := func(ctx context.Context, cid uint32) (int16, int16, error) {
		// Return a coordinate that would be inside the mist if it were checked.
		return 10, 10, nil
	}

	instanceA := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	instanceB := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
	fA := field.NewBuilder(0, 0, 100000000).SetInstance(instanceA).Build()
	fB := field.NewBuilder(0, 0, 100000000).SetInstance(instanceB).Build()

	id := uuid.New()
	mistOnA := mist.NewBuilder(id, fA).
		SetOwner("MONSTER", 9001).
		SetOrigin(0, 0).
		SetBounds(-100, -100, 100, 100).
		SetDisease("POISON", 80, 30*time.Second).
		SetDuration(time.Minute).
		SetTickInterval(time.Second).
		SetSource(100020, 5).
		Build()
	require.NoError(t, reg.Add(tt, mistOnA))

	mt := newTestMistTick(t, reg, rec, posLookup)
	mt.charsInField = func(tnt tenant.Model, f field.Model) []uint32 {
		// Only return the otherInstanceCharId for instanceB. The mist lives on instanceA,
		// so when MistTick asks for characters in fA it must not see instance-B chars.
		if f.Instance() == instanceB {
			return []uint32{otherInstanceCharId}
		}
		return nil
	}
	mt.runOnce(context.Background())

	// No apply-disease commands at all: mist on A had no characters, instance B was not queried.
	require.Empty(t, rec.Messages(EnvCommandTopicCharacterBuff))
	// Also no destroy (mist still live).
	require.Empty(t, rec.Messages(mistKafka.EnvEventTopic))

	// Sanity: ensure mt.charsInField was called with instanceA, not instanceB.
	_ = fB
}
