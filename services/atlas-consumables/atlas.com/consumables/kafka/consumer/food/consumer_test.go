package food

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	foodmsg "atlas-consumables/kafka/message/food"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

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

func setupCapturingProducer(t *testing.T) *writerRegistry {
	t.Helper()
	reg := newWriterRegistry()
	kafkaProducer.ResetInstance()
	kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(reg.factory()))
	t.Cleanup(kafkaProducer.ResetInstance)
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
// atlas-mounts Task 20 consumes, with the pinned heal of 30.
func TestTamingMobFedEventProviderShape(t *testing.T) {
	if foodmsg.RevitalizerTirednessHeal != 30 {
		t.Fatalf("expected pinned tiredness heal 30, got %d", foodmsg.RevitalizerTirednessHeal)
	}

	ev := foodmsg.Event{
		WorldId:       world.Id(2),
		CharacterId:   42,
		ItemId:        2260000,
		TirednessHeal: foodmsg.RevitalizerTirednessHeal,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	for _, k := range []string{"worldId", "characterId", "itemId", "tirednessHeal"} {
		if _, ok := decoded[k]; !ok {
			t.Fatalf("event JSON missing key %q; got %s", k, string(b))
		}
	}
	if int(decoded["tirednessHeal"].(float64)) != 30 {
		t.Fatalf("expected tirednessHeal 30, got %v", decoded["tirednessHeal"])
	}
}
