package pickup

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	mbmsg "atlas-consumables/kafka/message/monsterbook"
	pickupmsg "atlas-consumables/kafka/message/pickup"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// emitted captures everything this package's tests produce to Kafka. Installed
// once for the package; individual tests call emitted.Reset() rather than
// reinstalling the manager (DOM-24(e)).
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	os.Setenv(string(mbmsg.EnvCommandTopic), string(mbmsg.EnvCommandTopic))
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
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
	emitted.Reset()
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
	msgs := emitted.Messages(string(mbmsg.EnvCommandTopic))
	if len(msgs) != 1 {
		t.Fatalf("expected 1 monster book message, got %d", len(msgs))
	}

	var out mbmsg.Command[mbmsg.CardPickedUpBody]
	if err := json.Unmarshal(msgs[0].Value, &out); err != nil {
		t.Fatalf("unmarshal emitted command: %v", err)
	}
	if out.Type != mbmsg.CommandTypeCardPickedUp {
		t.Fatalf("expected Type=%q, got %q", mbmsg.CommandTypeCardPickedUp, out.Type)
	}
	if out.Body.CardId != cmd.ItemId {
		t.Fatalf("expected CardId=%d, got %d", cmd.ItemId, out.Body.CardId)
	}
	if out.Body.Source != mbmsg.SourceDropPickup {
		t.Fatalf("expected Source=%q, got %q", mbmsg.SourceDropPickup, out.Body.Source)
	}
	if out.EventId != txnId {
		t.Fatalf("expected EventId=%s, got %s", txnId, out.EventId)
	}
	if out.CharacterId != cmd.CharacterId {
		t.Fatalf("expected CharacterId=%d, got %d", cmd.CharacterId, out.CharacterId)
	}
}

func TestHandlePickupNonCardItemSkips(t *testing.T) {
	emitted.Reset()
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

	if got := len(emitted.Messages(string(mbmsg.EnvCommandTopic))); got > 0 {
		t.Fatalf("expected no monster book emission for non-card item, got %d messages", got)
	}
}

func TestHandlePickupWrongTypeSkips(t *testing.T) {
	emitted.Reset()
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

	if got := len(emitted.Messages(string(mbmsg.EnvCommandTopic))); got > 0 {
		t.Fatalf("expected no monster book emission for wrong type, got %d messages", got)
	}
}
