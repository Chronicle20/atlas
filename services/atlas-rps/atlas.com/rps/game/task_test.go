package game_test

import (
	"atlas-rps/game"
	"atlas-rps/kafka/message/rps"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	rpsSaga "atlas-rps/kafka/message/saga"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// sweepCapturingWriter records every WriteMessages call so sweep tests can
// verify what was (or was not) emitted per topic. Mirrors
// kafka/consumer/rps/consumer_test.go's capturingWriter.
type sweepCapturingWriter struct {
	topic string
	mu    sync.Mutex
	msgs  []kafka.Message
}

func (w *sweepCapturingWriter) Topic() string { return w.topic }
func (w *sweepCapturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgs = append(w.msgs, msgs...)
	return nil
}
func (w *sweepCapturingWriter) Close() error { return nil }
func (w *sweepCapturingWriter) Messages() []kafka.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]kafka.Message, len(w.msgs))
	copy(out, w.msgs)
	return out
}

type sweepWriterRegistry struct {
	mu      sync.Mutex
	writers map[string]*sweepCapturingWriter
}

func newSweepWriterRegistry() *sweepWriterRegistry {
	return &sweepWriterRegistry{writers: map[string]*sweepCapturingWriter{}}
}

func (r *sweepWriterRegistry) factory() kafkaProducer.WriterFactory {
	return func(topicName string) kafkaProducer.Writer {
		r.mu.Lock()
		defer r.mu.Unlock()
		w := &sweepCapturingWriter{topic: topicName}
		r.writers[topicName] = w
		return w
	}
}

func (r *sweepWriterRegistry) get(topicName string) *sweepCapturingWriter {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writers[topicName]
}

// setupSweepCapturingProducer installs a capturing producer manager on top
// of the TestMain no-op floor, so the sweep task's real producer.ProviderImpl
// emission can be inspected per topic (including confirming the saga command
// topic - kafka/message/saga.EnvCommandTopic - never receives a message, since
// a swept session must never trigger a payout).
func setupSweepCapturingProducer(t *testing.T) *sweepWriterRegistry {
	t.Helper()
	reg := newSweepWriterRegistry()
	kafkaProducer.ResetInstance()
	kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(reg.factory()))
	return reg
}

func decodeSweepGameEnded(t *testing.T, msg kafka.Message) rps.Event[rps.GameEndedEventBody] {
	t.Helper()
	var e rps.Event[rps.GameEndedEventBody]
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	return e
}

func TestNewSweepTask(t *testing.T) {
	task := game.NewSweepTask(testLogger(), 100*time.Millisecond)
	assert.NotNil(t, task)
}

func TestSweepTask_SleepTime(t *testing.T) {
	interval := 250 * time.Millisecond
	task := game.NewSweepTask(testLogger(), interval)
	assert.Equal(t, interval, task.SleepTime())
}

// TestSweepTask_Run_NoExpiredSessions verifies a session well within its TTL
// is left untouched and nothing is emitted.
func TestSweepTask_Run_NoExpiredSessions(t *testing.T) {
	setupRegistryTest(t)
	reg := setupSweepCapturingProducer(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2001)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m := game.NewModelBuilder(ten).
		SetCharacterId(characterId).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9020000).
		SetStatus(game.StatusOpen).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	task := game.NewSweepTask(testLogger(), 50*time.Millisecond)
	task.Run()

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.True(t, found, "session within TTL should not be swept")

	if w := reg.get(rps.EnvEventTopic); w != nil {
		assert.Len(t, w.Messages(), 0)
	}
	if w := reg.get(rpsSaga.EnvCommandTopic); w != nil {
		assert.Len(t, w.Messages(), 0)
	}
}

// TestSweepTask_Run_DisposesExpiredSessionWithNoPayout is the core Task 12
// sweeper assertion: a session advanced past its TTL is popped from the
// registry and disposed with NO payout saga submitted, regardless of any
// prize the session's rung might otherwise resolve to.
func TestSweepTask_Run_DisposesExpiredSessionWithNoPayout(t *testing.T) {
	setupRegistryTest(t)
	reg := setupSweepCapturingProducer(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2002)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m := game.NewModelBuilder(ten).
		SetCharacterId(characterId).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9020000).
		SetRung(2).
		SetStatus(game.StatusAwaitingDecision).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	// Advance the clock well past the registry's TTL.
	now = now.Add(10 * time.Minute)
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	task := game.NewSweepTask(testLogger(), 50*time.Millisecond)
	task.Run()

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "expired session should be popped from the registry")

	// No payout saga must ever be produced by the sweeper.
	if w := reg.get(rpsSaga.EnvCommandTopic); w != nil {
		assert.Len(t, w.Messages(), 0, "sweeper must never submit a payout saga")
	}

	// The sweep disposes with a GameEnded{disconnected} event, mirroring
	// Processor.Dispose - no granted prize.
	w := reg.get(rps.EnvEventTopic)
	require.NotNil(t, w, "expected the RPS event topic writer to have been used")
	msgs := w.Messages()
	require.Len(t, msgs, 1)
	ended := decodeSweepGameEnded(t, msgs[0])
	assert.Equal(t, rps.ReasonDisconnected, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize)
}

// TestSweepTask_Run_MultiTenantSweepsAcrossAllTracked verifies the sweeper
// reclaims expired sessions across every tracked tenant, not just the
// tenant on the ambient context.
func TestSweepTask_Run_MultiTenantSweepsAcrossAllTracked(t *testing.T) {
	setupRegistryTest(t)
	reg := setupSweepCapturingProducer(t)
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)
	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m1 := game.NewModelBuilder(ten1).SetCharacterId(3001).SetWorldId(0).SetChannelId(1).SetNpcId(9020000).SetStatus(game.StatusOpen).MustBuild()
	game.GetRegistry().Put(ctx1, m1)
	m2 := game.NewModelBuilder(ten2).SetCharacterId(3002).SetWorldId(0).SetChannelId(1).SetNpcId(9020000).SetStatus(game.StatusOpen).MustBuild()
	game.GetRegistry().Put(ctx2, m2)

	now = now.Add(10 * time.Minute)
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	task := game.NewSweepTask(testLogger(), 50*time.Millisecond)
	task.Run()

	_, found1 := game.GetRegistry().Get(ctx1, 3001)
	assert.False(t, found1)
	_, found2 := game.GetRegistry().Get(ctx2, 3002)
	assert.False(t, found2)

	w := reg.get(rps.EnvEventTopic)
	require.NotNil(t, w)
	assert.Len(t, w.Messages(), 2, "expected a GameEnded(disconnected) for each swept session")
}
