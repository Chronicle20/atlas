package list

import (
	"atlas-buddies/buddy"
	"atlas-buddies/kafka/message"
	list2 "atlas-buddies/kafka/message/list"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	character2 "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain installs a no-op producer writer so any DIRECT-path emits
// (rejectEmit closures fired outside the outbox-bound buffer, per the D7
// fix below) succeed instantly instead of retrying against an unreachable
// broker for ~42s (see producertest package doc).
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}

// capturingWriter records every message written to it, keyed by resolved
// topic name, instead of discarding (producertest.NoopWriter). Used to
// inspect what the DIRECT producer path actually sends.
type capturingWriter struct {
	topic string
	mu    *sync.Mutex
	msgs  *map[string][]kafka.Message
}

func (w capturingWriter) Topic() string { return w.topic }

func (w capturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	(*w.msgs)[w.topic] = append((*w.msgs)[w.topic], msgs...)
	return nil
}

func (w capturingWriter) Close() error { return nil }

// installCapturingProducer swaps the process-wide producer manager singleton
// for one that records messages instead of discarding them, returning the
// captured-messages map and a restore func that must be deferred to put the
// TestMain-installed no-op writer back for subsequent tests.
func installCapturingProducer() (*map[string][]kafka.Message, func()) {
	var mu sync.Mutex
	captured := make(map[string][]kafka.Message)
	kafkaproducer.ResetInstance()
	kafkaproducer.GetManager(kafkaproducer.ConfigWriterFactory(func(topicName string) kafkaproducer.Writer {
		return capturingWriter{topic: topicName, mu: &mu, msgs: &captured}
	}))
	return &captured, func() {
		producertest.InstallNoop()
	}
}

func testTenant() tenant.Model {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return te
}

func testProcessorLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

// D7-fix regression tests for RequestDeleteBuddy: a rejection emitted while
// the inner tx rolls back must not ride into the caller-supplied (outbox-
// bound in production) mb; it must be fired on the DIRECT producer path
// instead. See the recipe's "Failure-path pitfalls" section and the
// analogous fix in atlas-inventory (b820a3db7).

// TestRequestDeleteBuddyMissingListRoutesRejectDirect exercises the first
// failure branch of RequestDeleteBuddy (GetByCharacterId fails because the
// character has no buddy list at all, a fast, deterministic, DB-only
// failure that needs no external service mocking).
func TestRequestDeleteBuddyMissingListRoutesRejectDirect(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	db := setupProcessorTestDB(t)
	l := testProcessorLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)

	characterId := uint32(500)
	targetId := uint32(501)
	worldId := world.Id(0)

	mb := message.NewBuffer()
	err := NewProcessor(l, ctx, db).RequestDeleteBuddy(mb)(characterId, worldId, targetId)
	if err != nil {
		t.Fatalf("RequestDeleteBuddy is expected to swallow the tx error and return nil, got: %v", err)
	}

	// D7: the inner tx rolled back (no buddy list exists for characterId),
	// so the rejection ERROR status event must NOT land in the caller-
	// supplied (outbox-bound in production) buffer.
	events := mb.GetAll()
	if len(events) != 0 {
		t.Fatalf("expected caller-supplied buffer to be empty on a rolled-back tx (D7), got: %#v", events)
	}

	// The rejection must instead have been fired on the DIRECT producer
	// path.
	msgs := (*captured)[list2.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 direct-path message on topic %s, got %d", list2.EnvStatusEventTopic, len(msgs))
	}
	var ev list2.StatusEvent[list2.ErrorStatusEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("failed to unmarshal direct-path message: %v", err)
	}
	if ev.Type != list2.StatusEventTypeError {
		t.Fatalf("expected direct-path event type %s, got %s", list2.StatusEventTypeError, ev.Type)
	}
	if ev.CharacterId != character2.Id(characterId) {
		t.Fatalf("expected direct-path event characterId %d, got %d", characterId, ev.CharacterId)
	}
}

// TestRequestDeleteBuddySuccessMergesIntoCallerBuffer guards the merge-on-
// success side of the D7 scratch-buffer fix: when the inner tx commits, the
// BUDDY_REMOVED event (buffered on a scratch innerMb, per the fix) must
// still land in the caller-supplied (outbox-bound in production) mb, and no
// direct-path emission should occur.
func TestRequestDeleteBuddySuccessMergesIntoCallerBuffer(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	db := setupProcessorTestDB(t)
	l := testProcessorLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)

	// setupProcessorTestDB creates the buddies table via raw SQL without a
	// tenant_id column (pre-dating buddy.Entity's TenantId field); add it so
	// the tenant-scoped read/preload used by GetByCharacterId can see the
	// row this test inserts below.
	if err := db.AutoMigrate(&buddy.Entity{}); err != nil {
		t.Fatalf("failed to migrate buddy.Entity: %v", err)
	}

	characterId := uint32(510)
	targetId := uint32(511)
	worldId := world.Id(0)

	// characterId's own list, with targetId already a buddy on it.
	characterEntity := Entity{
		TenantId:    te.Id(),
		Id:          uuid.New(),
		CharacterId: characterId,
		Capacity:    20,
	}
	if err := db.Create(&characterEntity).Error; err != nil {
		t.Fatalf("failed to create character list entity: %v", err)
	}
	if err := db.Create(&buddy.Entity{
		CharacterId:   targetId,
		ListId:        characterEntity.Id,
		TenantId:      te.Id(),
		Group:         "Default Group",
		CharacterName: "Target",
		ChannelId:     -1,
	}).Error; err != nil {
		t.Fatalf("failed to create buddy row: %v", err)
	}

	// targetId's own (empty) list, required by updateBuddyChannel's lookup.
	targetEntity := Entity{
		TenantId:    te.Id(),
		Id:          uuid.New(),
		CharacterId: targetId,
		Capacity:    20,
	}
	if err := db.Create(&targetEntity).Error; err != nil {
		t.Fatalf("failed to create target list entity: %v", err)
	}

	mb := message.NewBuffer()
	err := NewProcessor(l, ctx, db).RequestDeleteBuddy(mb)(characterId, worldId, targetId)
	if err != nil {
		t.Fatalf("expected no error on successful delete, got: %v", err)
	}

	events := mb.GetAll()
	statusMsgs := events[list2.EnvStatusEventTopic]
	if len(statusMsgs) != 1 {
		t.Fatalf("expected exactly 1 status event in the caller-supplied buffer, got %d: %#v", len(statusMsgs), events)
	}
	var ev list2.StatusEvent[list2.BuddyRemovedStatusEventBody]
	if err := json.Unmarshal(statusMsgs[0].Value, &ev); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if ev.Type != list2.StatusEventTypeBuddyRemoved {
		t.Fatalf("expected event type %s, got %s", list2.StatusEventTypeBuddyRemoved, ev.Type)
	}

	// No rejection should have fired on the direct path for a successful call.
	if len(*captured) != 0 {
		t.Fatalf("expected no direct-path emissions on success, got: %#v", *captured)
	}
}

// TestAcceptInviteMissingListRoutesRejectDirect exercises AcceptInvite's
// first failure branch (GetByCharacterId fails because the accepting
// character has no buddy list). Same D7 shape as RequestDeleteBuddy above.
func TestAcceptInviteMissingListRoutesRejectDirect(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	db := setupProcessorTestDB(t)
	l := testProcessorLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)

	characterId := uint32(520)
	targetId := uint32(521)
	worldId := world.Id(0)

	mb := message.NewBuffer()
	err := NewProcessor(l, ctx, db).AcceptInvite(mb)(characterId, worldId, targetId)
	if err != nil {
		t.Fatalf("AcceptInvite is expected to swallow the tx error and return nil, got: %v", err)
	}

	events := mb.GetAll()
	if len(events) != 0 {
		t.Fatalf("expected caller-supplied buffer to be empty on a rolled-back tx (D7), got: %#v", events)
	}

	msgs := (*captured)[list2.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 direct-path message on topic %s, got %d", list2.EnvStatusEventTopic, len(msgs))
	}
	var ev list2.StatusEvent[list2.ErrorStatusEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("failed to unmarshal direct-path message: %v", err)
	}
	if ev.Type != list2.StatusEventTypeError {
		t.Fatalf("expected direct-path event type %s, got %s", list2.StatusEventTypeError, ev.Type)
	}
}
