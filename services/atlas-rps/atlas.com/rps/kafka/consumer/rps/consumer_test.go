package rps

import (
	"context"
	"sync"
	"testing"

	"atlas-rps/game"
	rpsMsg "atlas-rps/kafka/message/rps"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingWriter records every WriteMessages call so tests can verify what
// the rps consumer emitted (or did not emit).
type capturingWriter struct {
	topic string
	mu    sync.Mutex
	msgs  []kafka.Message
}

func (w *capturingWriter) Topic() string { return w.topic }
func (w *capturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgs = append(w.msgs, msgs...)
	return nil
}
func (w *capturingWriter) Close() error { return nil }
func (w *capturingWriter) Messages() []kafka.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]kafka.Message, len(w.msgs))
	copy(out, w.msgs)
	return out
}

type writerRegistry struct {
	mu      sync.Mutex
	writers map[string]*capturingWriter
}

func newWriterRegistry() *writerRegistry {
	return &writerRegistry{writers: map[string]*capturingWriter{}}
}

func (r *writerRegistry) factory() kafkaProducer.WriterFactory {
	return func(topicName string) kafkaProducer.Writer {
		r.mu.Lock()
		defer r.mu.Unlock()
		w := &capturingWriter{topic: topicName}
		r.writers[topicName] = w
		return w
	}
}

func (r *writerRegistry) get(topicName string) *capturingWriter {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writers[topicName]
}

// setupCapturingProducer installs a capturing producer manager on top of the
// TestMain no-op floor. It deliberately does not reset the singleton in
// cleanup (DOM-24(e)); the TestMain floor remains the baseline and the last
// installed capturing manager wins.
func setupCapturingProducer(t *testing.T) *writerRegistry {
	t.Helper()
	reg := newWriterRegistry()
	kafkaProducer.ResetInstance()
	kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(reg.factory()))
	return reg
}

// setupRegistry initializes the game.Registry singleton against a miniredis
// instance, isolated per test.
func setupRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	game.InitRegistry(client)
}

func tenantCtx(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), ten), ten
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// fixedThrows returns a game.ThrowSource that always plays back t.
func fixedThrows(t game.Throw) game.ThrowSource {
	return func() game.Throw { return t }
}

// stubLadder is a minimal single-rung ladder so a Select win can resolve a
// LadderProvider without a real configuration REST server.
func stubLadderProvider() game.LadderProvider {
	return func() (game.Ladder, error) {
		return game.Ladder{}, nil
	}
}

// stubSagaSubmitter is a no-op game.SagaSubmitter for handler tests that
// don't exercise the payout path (Collect); it must still be non-nil to
// satisfy NewProcessorWithLadder.
func stubSagaSubmitter() game.SagaSubmitter {
	return func(sharedsaga.Saga) error { return nil }
}

// withStubProcessor overrides the package-level newProcessor seam for the
// duration of the test, restoring the original on cleanup.
func withStubProcessor(t *testing.T, throwSource game.ThrowSource) {
	t.Helper()
	orig := newProcessor
	newProcessor = func(l logrus.FieldLogger, ctx context.Context) game.Processor {
		return game.NewProcessorWithLadder(l, ctx, throwSource, stubLadderProvider(), stubSagaSubmitter())
	}
	t.Cleanup(func() { newProcessor = orig })
}

func openSession(t *testing.T, ctx context.Context, ten tenant.Model, characterId uint32) {
	t.Helper()
	m := game.NewModelBuilder(ten).
		SetCharacterId(characterId).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9020000).
		SetStatus(game.StatusOpen).
		MustBuild()
	game.GetRegistry().Put(ctx, m)
}

// TestHandleBeginCommand_InvokesProcessorAndEmits verifies a BEGIN command
// against an open session transitions it to StatusAwaitingSelect and
// emits a RoundStarted event.
func TestHandleBeginCommand_InvokesProcessorAndEmits(t *testing.T) {
	setupRegistry(t)
	reg := setupCapturingProducer(t)
	ctx, ten := tenantCtx(t)
	characterId := uint32(5010)
	openSession(t, ctx, ten, characterId)

	withStubProcessor(t, fixedThrows(game.ThrowScissors))

	cmd := rpsMsg.Command[rpsMsg.BeginCommandBody]{
		CharacterId: characterId,
		WorldId:     0,
		ChannelId:   1,
		Type:        rpsMsg.CommandTypeBegin,
		Body:        rpsMsg.BeginCommandBody{},
	}

	handleBeginCommand(testLogger(), ctx, cmd)

	updated, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found, "session should still be present after BEGIN")
	assert.Equal(t, game.StatusAwaitingSelect, updated.Status())

	w := reg.get(rpsMsg.EnvEventTopic)
	require.NotNil(t, w, "expected the RPS event topic writer to have been used")
	require.Len(t, w.Messages(), 1, "expected exactly one RoundStarted event")
}

// TestHandleBeginCommand_WrongTypeSkips verifies the handler ignores commands
// whose Type is not BEGIN.
func TestHandleBeginCommand_WrongTypeSkips(t *testing.T) {
	setupRegistry(t)
	reg := setupCapturingProducer(t)
	ctx, ten := tenantCtx(t)
	characterId := uint32(5011)
	openSession(t, ctx, ten, characterId)
	withStubProcessor(t, fixedThrows(game.ThrowScissors))

	cmd := rpsMsg.Command[rpsMsg.BeginCommandBody]{
		CharacterId: characterId,
		Type:        "OTHER",
		Body:        rpsMsg.BeginCommandBody{},
	}

	handleBeginCommand(testLogger(), ctx, cmd)

	updated, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found)
	assert.Equal(t, game.StatusOpen, updated.Status(), "session must be untouched for a wrong-type command")
	assert.Nil(t, reg.get(rpsMsg.EnvEventTopic), "no event should be emitted for a wrong-type command")
}

// TestHandleSelectCommand_InvokesProcessorAndEmits verifies a SELECT command
// against an open session is applied by the game.Processor (registry state
// changes) and buffers/emits a RoundResult event via the real
// NewProcessorWithLadder wiring the handler builds.
func TestHandleSelectCommand_InvokesProcessorAndEmits(t *testing.T) {
	setupRegistry(t)
	reg := setupCapturingProducer(t)
	ctx, ten := tenantCtx(t)
	characterId := uint32(5001)
	openSession(t, ctx, ten, characterId)

	// Rock beats Scissors: forces a deterministic win, driving the session to
	// rung 1 / StatusAwaitingDecision.
	withStubProcessor(t, fixedThrows(game.ThrowScissors))

	cmd := rpsMsg.Command[rpsMsg.SelectCommandBody]{
		CharacterId: characterId,
		WorldId:     0,
		ChannelId:   1,
		Type:        rpsMsg.CommandTypeSelect,
		Body:        rpsMsg.SelectCommandBody{Throw: byte(game.ThrowRock)},
	}

	handleSelectCommand(testLogger(), ctx, cmd)

	updated, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found, "session should still be present after a winning round")
	assert.Equal(t, game.StatusAwaitingDecision, updated.Status())
	assert.Equal(t, 1, updated.Rung())

	w := reg.get(rpsMsg.EnvEventTopic)
	require.NotNil(t, w, "expected the RPS event topic writer to have been used")
	msgs := w.Messages()
	require.Len(t, msgs, 1, "expected exactly one RoundResult event")
}

// TestHandleSelectCommand_WrongTypeSkips verifies the handler ignores
// commands whose Type is not SELECT and neither invokes the processor nor
// emits anything.
func TestHandleSelectCommand_WrongTypeSkips(t *testing.T) {
	setupRegistry(t)
	reg := setupCapturingProducer(t)
	ctx, ten := tenantCtx(t)
	characterId := uint32(5002)
	openSession(t, ctx, ten, characterId)
	withStubProcessor(t, fixedThrows(game.ThrowScissors))

	cmd := rpsMsg.Command[rpsMsg.SelectCommandBody]{
		CharacterId: characterId,
		Type:        "OTHER",
		Body:        rpsMsg.SelectCommandBody{Throw: byte(game.ThrowRock)},
	}

	handleSelectCommand(testLogger(), ctx, cmd)

	updated, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found)
	assert.Equal(t, game.StatusOpen, updated.Status(), "session must be untouched for a wrong-type command")

	if w := reg.get(rpsMsg.EnvEventTopic); w != nil {
		assert.Len(t, w.Messages(), 0, "expected no emission for a wrong-type command")
	}
}

// TestHandleQuitCommand_InvokesProcessorAndEmits verifies a QUIT command ends
// the session and emits a GameEnded(quit) event.
func TestHandleQuitCommand_InvokesProcessorAndEmits(t *testing.T) {
	setupRegistry(t)
	reg := setupCapturingProducer(t)
	ctx, ten := tenantCtx(t)
	characterId := uint32(5003)
	openSession(t, ctx, ten, characterId)
	withStubProcessor(t, fixedThrows(game.ThrowScissors))

	cmd := rpsMsg.Command[rpsMsg.QuitCommandBody]{
		CharacterId: characterId,
		Type:        rpsMsg.CommandTypeQuit,
	}

	handleQuitCommand(testLogger(), ctx, cmd)

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "session should be removed after Quit")

	w := reg.get(rpsMsg.EnvEventTopic)
	require.NotNil(t, w)
	msgs := w.Messages()
	require.Len(t, msgs, 1, "expected exactly one GameEnded(quit) event")
}
