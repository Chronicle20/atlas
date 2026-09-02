package handler

import (
	messageCashShop "atlas-channel/kafka/message/cashshop"
	"atlas-channel/session"
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// capturingWriter records every message written to it, keyed by resolved
// topic name, instead of discarding (producertest.NoopWriter) or hitting a
// real broker. Mirrors the pattern in
// services/atlas-fame/atlas.com/fame/fame/processor_test.go.
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
// for one that records messages instead of discarding or dialing a real
// broker, returning the captured-messages map and a restore func that must
// be deferred.
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

// newGachaponTestSession builds a session with a distinct account id and
// character id, so a test that mixed them up would be caught.
func newGachaponTestSession(t *testing.T, accountId uint32, characterId uint32) (session.Model, context.Context, func()) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetAccountId(sessionId, accountId)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	s = sp.SetField(sessionId, f)

	return s, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}

// gachaponCashIdBytes encodes the 8-byte little-endian cashId payload the
// client sends: CUICashItemGachapon::OnButtonClicked EncodeBuffer(&m_liItemSN, 8).
func gachaponCashIdBytes(cashId int64) []byte {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = byte(cashId >> (8 * i))
	}
	return b
}

func decodeOpenSurpriseCommand(t *testing.T, m kafka.Message) messageCashShop.Command[messageCashShop.OpenSurpriseCommandBody] {
	t.Helper()
	var cmd messageCashShop.Command[messageCashShop.OpenSurpriseCommandBody]
	if err := json.Unmarshal(m.Value, &cmd); err != nil {
		t.Fatalf("unmarshal OPEN_SURPRISE command: %v", err)
	}
	return cmd
}

// The edge does not own the locker: the handler must decode and forward,
// performing NO ownership, template, or capacity validation of its own.
// Every check lives in atlas-cashshop, which is the only service that can
// make them atomically with the grant.
func TestCashItemGachaponHandleProducesCommand(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	const accountId = uint32(777)
	const characterId = uint32(555)
	const cashId = int64(1234567890)

	s, ctx, cleanup := newGachaponTestSession(t, accountId, characterId)
	defer cleanup()

	raw := request.Request(gachaponCashIdBytes(cashId))
	reader := request.NewRequestReader(&raw, 0)

	handlerFunc := CashItemGachaponHandleFunc(logrus.New(), ctx, nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	msgs := (*captured)[string(messageCashShop.EnvCommandTopic)]
	if len(msgs) != 1 {
		t.Fatalf("OPEN_SURPRISE messages produced = %d, want 1", len(msgs))
	}

	cmd := decodeOpenSurpriseCommand(t, msgs[0])
	if cmd.Type != messageCashShop.CommandTypeOpenSurprise {
		t.Fatalf("command type = %q, want %q", cmd.Type, messageCashShop.CommandTypeOpenSurprise)
	}
	if cmd.CharacterId != characterId {
		t.Fatalf("characterId = %d, want %d (session)", cmd.CharacterId, characterId)
	}
	if cmd.Body.AccountId != accountId {
		t.Fatalf("accountId = %d, want %d (session)", cmd.Body.AccountId, accountId)
	}
	if cmd.Body.CashId != cashId {
		t.Fatalf("cashId = %d, want %d", cmd.Body.CashId, cashId)
	}
	if cmd.Body.TransactionId == uuid.Nil {
		t.Fatal("transactionId is nil, want a freshly minted uuid")
	}
}

// Two clicks must mint two distinct transaction ids, or the openings ledger
// would reject the second as a redelivery.
func TestCashItemGachaponHandleMintsDistinctTransactionIds(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	const accountId = uint32(777)
	const characterId = uint32(555)
	const cashId = int64(1234567890)

	s, ctx, cleanup := newGachaponTestSession(t, accountId, characterId)
	defer cleanup()

	handlerFunc := CashItemGachaponHandleFunc(logrus.New(), ctx, nil)

	raw1 := request.Request(gachaponCashIdBytes(cashId))
	reader1 := request.NewRequestReader(&raw1, 0)
	handlerFunc(s, &reader1, map[string]interface{}{})

	raw2 := request.Request(gachaponCashIdBytes(cashId))
	reader2 := request.NewRequestReader(&raw2, 0)
	handlerFunc(s, &reader2, map[string]interface{}{})

	msgs := (*captured)[string(messageCashShop.EnvCommandTopic)]
	if len(msgs) != 2 {
		t.Fatalf("OPEN_SURPRISE messages produced = %d, want 2", len(msgs))
	}

	cmd1 := decodeOpenSurpriseCommand(t, msgs[0])
	cmd2 := decodeOpenSurpriseCommand(t, msgs[1])
	if cmd1.Body.TransactionId == cmd2.Body.TransactionId {
		t.Fatalf("both clicks minted the same transactionId [%s]", cmd1.Body.TransactionId)
	}
}
