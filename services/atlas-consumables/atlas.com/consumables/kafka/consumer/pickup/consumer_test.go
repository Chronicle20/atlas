package pickup

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	mbmsg "atlas-consumables/kafka/message/monsterbook"
	pickupmsg "atlas-consumables/kafka/message/pickup"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// capturingWriter records every WriteMessages call so tests can verify
// what the pickup consumer emitted (or did not emit).
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

// writerRegistry records every Writer constructed by the manager so tests
// can inspect emissions per topic.
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

func TestHandlePickupCardItemEmitsMonsterBookCommand(t *testing.T) {
	reg := setupCapturingProducer(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	txnId := uuid.New()
	cmd := pickupmsg.Command{
		TenantId:      tid,
		CharacterId:   42,
		ItemId:        2380000,
		TransactionId: txnId,
		Type:          pickupmsg.CommandType,
	}

	handlePickup(logger, ctx, cmd)

	// EnvProvider falls back to the env var token name when unset.
	w := reg.get(mbmsg.EnvCommandTopic)
	if w == nil {
		t.Fatalf("no writer created for topic %s", mbmsg.EnvCommandTopic)
	}
	msgs := w.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 monster book message, got %d", len(msgs))
	}

	var emitted mbmsg.Command[mbmsg.CardPickedUpBody]
	if err := json.Unmarshal(msgs[0].Value, &emitted); err != nil {
		t.Fatalf("unmarshal emitted command: %v", err)
	}
	if emitted.Type != mbmsg.CommandTypeCardPickedUp {
		t.Fatalf("expected Type=%q, got %q", mbmsg.CommandTypeCardPickedUp, emitted.Type)
	}
	if emitted.Body.CardId != cmd.ItemId {
		t.Fatalf("expected CardId=%d, got %d", cmd.ItemId, emitted.Body.CardId)
	}
	if emitted.EventId != txnId {
		t.Fatalf("expected EventId=%s, got %s", txnId, emitted.EventId)
	}
	if emitted.CharacterId != cmd.CharacterId {
		t.Fatalf("expected CharacterId=%d, got %d", cmd.CharacterId, emitted.CharacterId)
	}
}

func TestHandlePickupNonCardItemSkips(t *testing.T) {
	reg := setupCapturingProducer(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cmd := pickupmsg.Command{
		TenantId:      tid,
		CharacterId:   42,
		ItemId:        2000000, // consumable, not a monster card
		TransactionId: uuid.New(),
		Type:          pickupmsg.CommandType,
	}

	handlePickup(logger, ctx, cmd)

	if w := reg.get(mbmsg.EnvCommandTopic); w != nil && len(w.Messages()) > 0 {
		t.Fatalf("expected no monster book emission for non-card item, got %d messages", len(w.Messages()))
	}
}

func TestHandlePickupWrongTypeSkips(t *testing.T) {
	reg := setupCapturingProducer(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cmd := pickupmsg.Command{
		TenantId:      tid,
		CharacterId:   42,
		ItemId:        2380000,
		TransactionId: uuid.New(),
		Type:          "OTHER",
	}

	handlePickup(logger, ctx, cmd)

	if w := reg.get(mbmsg.EnvCommandTopic); w != nil && len(w.Messages()) > 0 {
		t.Fatalf("expected no monster book emission for wrong type, got %d messages", len(w.Messages()))
	}
}
