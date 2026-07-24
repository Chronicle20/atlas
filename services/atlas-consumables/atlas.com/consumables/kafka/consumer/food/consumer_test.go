package food

import (
	"atlas-consumables/consumable"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	foodmsg "atlas-consumables/kafka/message/food"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain installs the shared no-op producer floor so any emit that escapes a
// test (e.g. a path that reaches the real producer singleton) discards instead
// of hanging on broker retries. Individual tests that need to inspect emissions
// install their own capturing manager on top of this floor; none of them reset
// the singleton in cleanup (DOM-24(e)).
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}

// capturingWriter records every WriteMessages call so tests can verify what the
// food consumer emitted (or did not emit).
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
// TestMain no-op floor. It deliberately does NOT register a cleanup that resets
// the producer singleton: DOM-24(e) forbids un-stubbing the shared singleton in
// test cleanup, which would leave later tests racing against an uninitialized
// manager. The TestMain floor remains the baseline; each test simply layers its
// own capturing manager and the last writer wins.
func setupCapturingProducer(t *testing.T) *writerRegistry {
	t.Helper()
	reg := newWriterRegistry()
	kafkaProducer.ResetInstance()
	kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(reg.factory()))
	return reg
}

func tenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// TestHandleRequestFeedWrongTypeSkips verifies the handler ignores commands
// whose Type is not REQUEST_FEED and emits nothing.
func TestHandleRequestFeedWrongTypeSkips(t *testing.T) {
	reg := setupCapturingProducer(t)
	ctx := tenantCtx(t, uuid.New())
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cmd := foodmsg.Command[foodmsg.RequestFeedBody]{
		WorldId:     world.Id(0),
		CharacterId: 42,
		Type:        "OTHER",
		Body:        foodmsg.RequestFeedBody{Slot: 3, ItemId: 2260000},
	}

	handleRequestFeed(logger, ctx, cmd)

	if w := reg.get(foodmsg.EnvEventTopic); w != nil && len(w.Messages()) > 0 {
		t.Fatalf("expected no emission for wrong type, got %d messages", len(w.Messages()))
	}
}

// TestHandleRequestFeedNon226Rejects verifies a non-revitalizer item is
// rejected at the classification gate: no reserve, no event.
func TestHandleRequestFeedNon226Rejects(t *testing.T) {
	reg := setupCapturingProducer(t)
	ctx := tenantCtx(t, uuid.New())
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cmd := foodmsg.Command[foodmsg.RequestFeedBody]{
		WorldId:     world.Id(0),
		CharacterId: 42,
		Type:        foodmsg.CommandRequestFeed,
		Body:        foodmsg.RequestFeedBody{Slot: 3, ItemId: 2000000}, // class 200, not 226
	}

	handleRequestFeed(logger, ctx, cmd)

	if w := reg.get(foodmsg.EnvEventTopic); w != nil && len(w.Messages()) > 0 {
		t.Fatalf("expected no emission for non-revitalizer item, got %d messages", len(w.Messages()))
	}
}

// TestTamingMobFedEventProviderShape verifies the event provider produces the
// exact cross-service contract (worldId/characterId/itemId/tirednessHeal) that
// atlas-mounts Task 20 consumes, with the pinned heal of 30. This is the event
// a class-226 (revitalizer) feed emits once its reservation commits.
func TestTamingMobFedEventProviderShape(t *testing.T) {
	if foodmsg.RevitalizerTirednessHeal != 30 {
		t.Fatalf("expected pinned tiredness heal 30, got %d", foodmsg.RevitalizerTirednessHeal)
	}

	const (
		wantWorld  = world.Id(2)
		wantCharId = uint32(42)
		wantItemId = uint32(2260000) // classification 226 revitalizer
	)

	msgs, err := consumable.TamingMobFedEventProvider(wantWorld, wantCharId, wantItemId, foodmsg.RevitalizerTirednessHeal)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 TamingMobFed message, got %d", len(msgs))
	}

	var ev foodmsg.Event
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if ev.WorldId != wantWorld {
		t.Fatalf("expected worldId %d, got %d", wantWorld, ev.WorldId)
	}
	if ev.CharacterId != wantCharId {
		t.Fatalf("expected characterId %d, got %d", wantCharId, ev.CharacterId)
	}
	if ev.ItemId != wantItemId {
		t.Fatalf("expected itemId %d, got %d", wantItemId, ev.ItemId)
	}
	if ev.TirednessHeal != 30 {
		t.Fatalf("expected tirednessHeal 30, got %d", ev.TirednessHeal)
	}

	// JSON wire keys must match the cross-service contract.
	var decoded map[string]any
	if err := json.Unmarshal(msgs[0].Value, &decoded); err != nil {
		t.Fatalf("unmarshal event map: %v", err)
	}
	for _, k := range []string{"worldId", "characterId", "itemId", "tirednessHeal"} {
		if _, ok := decoded[k]; !ok {
			t.Fatalf("event JSON missing key %q; got %s", k, string(msgs[0].Value))
		}
	}
}
