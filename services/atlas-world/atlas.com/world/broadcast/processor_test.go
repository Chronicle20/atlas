package broadcast_test

import (
	"atlas-world/broadcast"
	"atlas-world/test"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	goredis "github.com/redis/go-redis/v9"
)

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
