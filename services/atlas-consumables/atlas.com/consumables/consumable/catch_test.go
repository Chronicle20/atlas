package consumable

import (
	"context"
	"errors"
	"sync"
	"testing"

	consumable3 "atlas-consumables/data/consumable"
	compartmentmsg "atlas-consumables/kafka/message/compartment"
	consumablemsg "atlas-consumables/kafka/message/consumable"
	monsterMsg "atlas-consumables/kafka/message/monster"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// TestValidateCatchItem is the pre-reserve gate: only class-227 items with a
// non-zero create id may proceed. Everything else is rejected before the
// inventory is touched (FR-3.2).
func TestValidateCatchItem(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		create uint32
		wantOk bool
	}{
		{"a catch item with a reward", 2270000, 1902000, true},
		{"a catch item with no create id", 2270000, 0, false},
		{"a red potion", 2000000, 1902000, false},
		{"a revitalizer", 2260000, 1902000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ci, _ := consumable3.Extract(consumable3.RestModel{Id: tc.itemId, Create: tc.create})
			if got := validateCatchItem(tc.itemId, ci); got != tc.wantOk {
				t.Fatalf("validateCatchItem = %t, want %t", got, tc.wantOk)
			}
		})
	}
}

// TestCatchOutcomeDecision pins the two-way branch the resolution handler takes,
// separated from its Kafka plumbing so it is testable without a broker:
// success commits the reservation and grants the create item; failure cancels
// the reservation and grants nothing (FR-3.8, FR-3.9).
func TestCatchOutcomeDecision(t *testing.T) {
	cases := []struct {
		name       string
		success    bool
		wantCommit bool
		wantGrant  bool
		wantCancel bool
	}{
		{"a successful catch", true, true, true, false},
		{"a failed catch", false, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := catchOutcome(tc.success)
			if d.commit != tc.wantCommit || d.grant != tc.wantGrant || d.cancel != tc.wantCancel {
				t.Fatalf("catchOutcome(%t) = %+v", tc.success, d)
			}
		})
	}
}

// stubWriter is a minimal producer.Writer used only by
// TestConsumeCatchCancelsReservationWhenCommandProduceFails to force one
// specific topic's produce to fail while every other topic still records
// what was written, so the test can assert on the cancellation side-effects.
type stubWriter struct {
	topicName string
	fail      bool
	mu        *sync.Mutex
	store     map[string][]kafka.Message
}

func (w stubWriter) Topic() string { return w.topicName }

func (w stubWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	if w.fail {
		return errors.New("produce failed: broker unavailable")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.store[w.topicName] = append(w.store[w.topicName], msgs...)
	return nil
}

func (w stubWriter) Close() error { return nil }

// TestConsumeCatchCancelsReservationWhenCommandProduceFails proves the
// reservation-leak fix: if the CATCH command itself cannot be produced (the
// scenario here — broker unavailable), no CATCH_RESOLVED will ever arrive to
// drive catchResolutionHandler, so ConsumeCatch must cancel the reservation
// itself rather than leaving it to dangle forever with the client never
// unlocked.
func TestConsumeCatchCancelsReservationWhenCommandProduceFails(t *testing.T) {
	tid := uuid.New()
	ctx := cardTenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	var mu sync.Mutex
	store := map[string][]kafka.Message{}
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(func(topicName string) producer.Writer {
		return stubWriter{topicName: topicName, fail: topicName == monsterMsg.EnvCommandTopic, mu: &mu, store: store}
	}))
	t.Cleanup(func() {
		emitted = producertest.InstallCapturing()
	})

	const characterId = uint32(77)
	const monsterUniqueId = uint32(555)
	const itemId = item2.Id(2270000)
	const slot = int16(3)
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()

	consume := ConsumeCatch(f, monsterUniqueId, characterId, itemId, transactionId, slot)
	if err := consume(logger)(ctx); err == nil {
		t.Fatal("expected an error from the failed CATCH produce, got nil")
	}

	// The CATCH command itself must never have landed (the produce failed).
	if got := len(store[monsterMsg.EnvCommandTopic]); got != 0 {
		t.Fatalf("expected 0 CATCH commands recorded, got %d", got)
	}

	// The reservation must be cancelled — otherwise the item stays reserved
	// forever with no CATCH_RESOLVED ever coming to release it.
	if got := len(store[compartmentmsg.EnvCommandTopic]); got != 1 {
		t.Fatalf("expected 1 compartment command (the cancellation), got %d", got)
	}

	// The client must be unlocked via the same generic ERROR event every
	// other ItemConsumer's ConsumeError call uses.
	msgs := store[consumablemsg.EnvEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 consumable event (the unlock), got %d", len(msgs))
	}
}
