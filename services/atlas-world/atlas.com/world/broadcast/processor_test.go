package broadcast_test

import (
	"atlas-world/broadcast"
	bmessage "atlas-world/kafka/message/broadcast"
	"atlas-world/test"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// capturingWriter implements kproducer.Writer by recording every message
// written to it, so a test can install it as the process-wide producer
// manager's writer factory and then inspect exactly what Enqueue emitted —
// without a mock broker (no network, no partitions; a synchronous in-memory
// sink, same shape as producertest.NoopWriter but recording instead of
// discarding).
type capturingWriter struct {
	topicName string
	mu        sync.Mutex
	messages  []kafka.Message
}

func (w *capturingWriter) Topic() string { return w.topicName }

func (w *capturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages = append(w.messages, msgs...)
	return nil
}

func (w *capturingWriter) Close() error { return nil }

func (w *capturingWriter) snapshot() []kafka.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]kafka.Message, len(w.messages))
	copy(out, w.messages)
	return out
}

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	broadcast.InitRegistry(rc)
}

func setupProcessor(t *testing.T) (broadcast.Processor, func()) {
	t.Helper()
	setupTestRegistry(t)
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := broadcast.NewProcessor(logger, ctx)
	return processor, func() {}
}

func newEntry(characterId uint32, durationSeconds uint32) broadcast.Entry {
	return broadcast.Entry{
		Id:              uuid.New(),
		CharacterId:     characterId,
		DurationSeconds: durationSeconds,
		Payload: broadcast.Payload{
			SenderName: "Sender",
		},
	}
}

func TestEnqueue_OnIdleQueue_ActivatesImmediately(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	e := newEntry(1, 30)
	if err := processor.Enqueue(world.Id(1), broadcast.FamilyTV, e); err != nil {
		t.Fatalf("Enqueue() unexpected error: %v", err)
	}

	q, err := processor.GetQueue(world.Id(1), broadcast.FamilyTV)
	if err != nil {
		t.Fatalf("GetQueue() unexpected error: %v", err)
	}
	if q.Active == nil {
		t.Fatal("Active = nil, want the enqueued entry to have activated immediately")
	}
	if q.Active.CharacterId != e.CharacterId {
		t.Errorf("Active.CharacterId = %d, want %d", q.Active.CharacterId, e.CharacterId)
	}
	if q.Active.ActivatedAt.IsZero() {
		t.Error("Active.ActivatedAt should be stamped, got zero value")
	}
	if q.Active.ExpiresAt.IsZero() {
		t.Error("Active.ExpiresAt should be stamped, got zero value")
	}
	if len(q.Pending) != 0 {
		t.Errorf("len(Pending) = %d, want 0", len(q.Pending))
	}
}

func TestEnqueue_OnBusyQueue_AppendsToPending(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	first := newEntry(1, 60)
	if err := processor.Enqueue(world.Id(1), broadcast.FamilyTV, first); err != nil {
		t.Fatalf("Enqueue() first entry unexpected error: %v", err)
	}

	second := newEntry(2, 30)
	if err := processor.Enqueue(world.Id(1), broadcast.FamilyTV, second); err != nil {
		t.Fatalf("Enqueue() second entry unexpected error: %v", err)
	}

	q, err := processor.GetQueue(world.Id(1), broadcast.FamilyTV)
	if err != nil {
		t.Fatalf("GetQueue() unexpected error: %v", err)
	}
	if q.Active == nil {
		t.Fatal("Active = nil, want the first entry to remain active")
	}
	if q.Active.CharacterId != first.CharacterId {
		t.Errorf("Active.CharacterId = %d, want %d (first entry should stay active)", q.Active.CharacterId, first.CharacterId)
	}
	if len(q.Pending) != 1 {
		t.Fatalf("len(Pending) = %d, want 1", len(q.Pending))
	}
	if q.Pending[0].CharacterId != second.CharacterId {
		t.Errorf("Pending[0].CharacterId = %d, want %d", q.Pending[0].CharacterId, second.CharacterId)
	}

	// The pure WaitSeconds computation itself is covered exhaustively by
	// model_test.go; here we only assert that Enqueue leaves the queue in
	// the state WaitSeconds would be computed from (active + one pending).
	wait := q.WaitSeconds(time.Now())
	if wait == 0 {
		t.Error("WaitSeconds(now) = 0, want > 0 for a busy queue with one pending entry")
	}
}

func TestEnqueue_DifferentFamiliesAreIndependent(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	tv := newEntry(1, 30)
	avatar := newEntry(2, 10)

	if err := processor.Enqueue(world.Id(1), broadcast.FamilyTV, tv); err != nil {
		t.Fatalf("Enqueue(TV) unexpected error: %v", err)
	}
	if err := processor.Enqueue(world.Id(1), broadcast.FamilyAvatar, avatar); err != nil {
		t.Fatalf("Enqueue(AVATAR) unexpected error: %v", err)
	}

	tvQueue, err := processor.GetQueue(world.Id(1), broadcast.FamilyTV)
	if err != nil {
		t.Fatalf("GetQueue(TV) unexpected error: %v", err)
	}
	if tvQueue.Active == nil || tvQueue.Active.CharacterId != tv.CharacterId {
		t.Errorf("TV queue Active = %+v, want CharacterId %d", tvQueue.Active, tv.CharacterId)
	}

	avatarQueue, err := processor.GetQueue(world.Id(1), broadcast.FamilyAvatar)
	if err != nil {
		t.Fatalf("GetQueue(AVATAR) unexpected error: %v", err)
	}
	if avatarQueue.Active == nil || avatarQueue.Active.CharacterId != avatar.CharacterId {
		t.Errorf("AVATAR queue Active = %+v, want CharacterId %d", avatarQueue.Active, avatar.CharacterId)
	}
}

func TestSweepTenant_ExpiresActiveAndPromotesNext(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	// DurationSeconds: 0 so the entry's ExpiresAt equals its ActivatedAt;
	// any later real-clock read is already >= ExpiresAt, so it is expired
	// without needing a real sleep beyond clock resolution.
	first := newEntry(1, 0)
	second := newEntry(2, 45)

	if err := processor.Enqueue(world.Id(7), broadcast.FamilyTV, first); err != nil {
		t.Fatalf("Enqueue() first entry unexpected error: %v", err)
	}
	if err := processor.Enqueue(world.Id(7), broadcast.FamilyTV, second); err != nil {
		t.Fatalf("Enqueue() second entry unexpected error: %v", err)
	}

	// Guarantee the wall clock has advanced past the zero-duration entry's
	// ExpiresAt before sweeping.
	time.Sleep(2 * time.Millisecond)

	if err := processor.SweepTenant(); err != nil {
		t.Fatalf("SweepTenant() unexpected error: %v", err)
	}

	q, err := processor.GetQueue(world.Id(7), broadcast.FamilyTV)
	if err != nil {
		t.Fatalf("GetQueue() unexpected error: %v", err)
	}
	if q.Active == nil {
		t.Fatal("Active = nil, want the second (pending) entry to have been promoted")
	}
	if q.Active.CharacterId != second.CharacterId {
		t.Errorf("Active.CharacterId = %d, want %d (second entry should have been promoted)", q.Active.CharacterId, second.CharacterId)
	}
	if len(q.Pending) != 0 {
		t.Errorf("len(Pending) = %d, want 0", len(q.Pending))
	}
}

func TestSweepTenant_ActiveNotYetExpired_NoOp(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	e := newEntry(1, 300)
	if err := processor.Enqueue(world.Id(3), broadcast.FamilyTV, e); err != nil {
		t.Fatalf("Enqueue() unexpected error: %v", err)
	}

	if err := processor.SweepTenant(); err != nil {
		t.Fatalf("SweepTenant() unexpected error: %v", err)
	}

	q, err := processor.GetQueue(world.Id(3), broadcast.FamilyTV)
	if err != nil {
		t.Fatalf("GetQueue() unexpected error: %v", err)
	}
	if q.Active == nil {
		t.Fatal("Active = nil, want the not-yet-expired entry to remain active")
	}
	if q.Active.CharacterId != e.CharacterId {
		t.Errorf("Active.CharacterId = %d, want %d (unexpired entry should be untouched)", q.Active.CharacterId, e.CharacterId)
	}
}

func TestSweepTenant_OnEmptyTenant_NoOp(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	if err := processor.SweepTenant(); err != nil {
		t.Fatalf("SweepTenant() on a tenant with no queues unexpected error: %v", err)
	}
}

func TestEnqueue_ConcurrentCASConflict_BothLand(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	var wg sync.WaitGroup
	errs := make([]error, 2)
	entries := []broadcast.Entry{newEntry(1, 30), newEntry(2, 30)}

	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			errs[idx] = processor.Enqueue(world.Id(9), broadcast.FamilyAvatar, entries[idx])
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("Enqueue() goroutine %d unexpected error: %v", i, err)
		}
	}

	q, err := processor.GetQueue(world.Id(9), broadcast.FamilyAvatar)
	if err != nil {
		t.Fatalf("GetQueue() unexpected error: %v", err)
	}

	total := len(q.Pending)
	if q.Active != nil {
		total++
	}
	if total != 2 {
		t.Fatalf("total entries in queue = %d, want 2 (both concurrent Enqueue calls should land: got Active=%v Pending=%d)", total, q.Active, len(q.Pending))
	}

	seen := map[uint32]bool{}
	if q.Active != nil {
		seen[q.Active.CharacterId] = true
	}
	for _, p := range q.Pending {
		seen[p.CharacterId] = true
	}
	for _, e := range entries {
		if !seen[e.CharacterId] {
			t.Errorf("entry for characterId %d missing from final queue state", e.CharacterId)
		}
	}
}

// TestEnqueue_StartedPayloadFieldMapping exercises the real code path that
// builds a STARTED event from an activated Entry — processor.Enqueue ->
// (unexported) startedPayload() -> bproducer.StartedStatusEventProvider —
// and asserts every one of the 11 mapped fields lands in the matching
// StatusEvent slot. startedPayload() hand-maps fields of the same type
// (SenderName/SenderMedal are both string; ReceiverName likewise); a future
// refactor that swaps two of them would leave go build/go vet green with no
// other test failing. This test installs a capturingWriter (in-process
// recording sink, not a mock broker) as the producer manager's writer for
// the duration of the test, so it can inspect the actual kafka.Message
// Enqueue produced instead of discarding it via producertest.NoopWriter.
func TestEnqueue_StartedPayloadFieldMapping(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	cw := &capturingWriter{}
	kproducer.ResetInstance()
	kproducer.GetManager(kproducer.ConfigWriterFactory(func(topicName string) kproducer.Writer {
		cw.topicName = topicName
		return cw
	}))
	defer producertest.InstallNoop() // restore the process-wide noop manager for subsequent tests

	receiverLook := sharedsaga.AvatarSnapshot{
		Gender:       1,
		SkinColor:    4,
		Face:         21000,
		Hair:         31000,
		Equips:       map[int16]uint32{-1: 1002141},
		MaskedEquips: map[int16]uint32{-101: 1002999},
		Pets:         map[int8]uint32{1: 5000002},
	}
	e := broadcast.Entry{
		Id:              uuid.New(),
		CharacterId:     555,
		DurationSeconds: 30,
		Payload: broadcast.Payload{
			ChannelId:     2,
			SenderName:    "sender-name",
			SenderMedal:   "sender-medal",
			Messages:      []string{"line-one", "line-two"},
			WhispersOn:    true,
			ItemId:        5390000,
			TvMessageType: "HEART",
			SenderLook: sharedsaga.AvatarSnapshot{
				Gender:       0,
				SkinColor:    3,
				Face:         20000,
				Hair:         30000,
				Equips:       map[int16]uint32{-1: 1002140},
				MaskedEquips: map[int16]uint32{-5: 1040002},
				Pets:         map[int8]uint32{0: 5000001},
			},
			ReceiverName: "receiver-name",
			ReceiverLook: &receiverLook,
		},
	}

	if err := processor.Enqueue(world.Id(11), broadcast.FamilyTV, e); err != nil {
		t.Fatalf("Enqueue() unexpected error: %v", err)
	}

	var started *bmessage.StatusEvent
	for _, m := range cw.snapshot() {
		var se bmessage.StatusEvent
		if err := json.Unmarshal(m.Value, &se); err != nil {
			t.Fatalf("json.Unmarshal(message.Value) unexpected error: %v", err)
		}
		if se.Type == bmessage.StatusTypeStarted {
			started = &se
			break
		}
	}
	if started == nil {
		t.Fatalf("no STARTED event captured; got %d message(s)", len(cw.snapshot()))
	}

	require.Equal(t, e.CharacterId, started.CharacterId)
	require.Equal(t, e.DurationSeconds, started.TotalWaitSeconds)
	require.Equal(t, e.Payload.ChannelId, started.ChannelId)
	require.Equal(t, e.Payload.SenderName, started.SenderName)
	require.Equal(t, e.Payload.SenderMedal, started.SenderMedal)
	require.Equal(t, e.Payload.Messages, started.Messages)
	require.Equal(t, e.Payload.WhispersOn, started.WhispersOn)
	require.Equal(t, e.Payload.ItemId, started.ItemId)
	require.Equal(t, e.Payload.TvMessageType, started.TvMessageType)
	require.Equal(t, e.Payload.SenderLook, started.SenderLook)
	require.Equal(t, e.Payload.ReceiverName, started.ReceiverName)
	require.NotNil(t, started.ReceiverLook)
	require.Equal(t, *e.Payload.ReceiverLook, *started.ReceiverLook)
}
