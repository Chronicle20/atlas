package consumable

import (
	"context"
	"encoding/json"
	"testing"

	compartmentmsg "atlas-consumables/kafka/message/compartment"
	mbmsg "atlas-consumables/kafka/message/monsterbook"

	"github.com/google/uuid"

	"github.com/sirupsen/logrus"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func cardTenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// A monster card used out of the inventory must both consume the reserved item
// and register the card. Before this branch existed the request fell through to
// ConsumeBare, which consumed the card and emitted nothing — the card was
// destroyed without ever reaching the monster book.
func TestConsumeMonsterCardConsumesAndEmitsCardPickedUp(t *testing.T) {
	emitted.Reset()
	tid := uuid.New()
	ctx := cardTenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	txnId := uuid.New()
	const characterId = uint32(12)
	const cardId = item2.Id(2380003)
	const slot = int16(8)

	consume := ConsumeMonsterCard(txnId, characterId, slot, cardId, inventory2.TypeValueUse)
	if err := consume(logger)(ctx); err != nil {
		t.Fatalf("ConsumeMonsterCard: %v", err)
	}

	// The reservation must be committed — otherwise the card stays in the
	// inventory and the book gains a card the character still holds.
	if got := len(emitted.Messages(string(compartmentmsg.EnvCommandTopic))); got != 1 {
		t.Fatalf("expected 1 compartment command, got %d", got)
	}

	msgs := emitted.Messages(string(mbmsg.EnvCommandTopic))
	if len(msgs) != 1 {
		t.Fatalf("expected 1 monster book message, got %d", len(msgs))
	}

	var cmd mbmsg.Command[mbmsg.CardPickedUpBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal emitted command: %v", err)
	}
	if cmd.Type != mbmsg.CommandTypeCardPickedUp {
		t.Fatalf("expected Type=%q, got %q", mbmsg.CommandTypeCardPickedUp, cmd.Type)
	}
	if cmd.TenantId != tid {
		t.Fatalf("expected TenantId=%s, got %s", tid, cmd.TenantId)
	}
	if cmd.CharacterId != characterId {
		t.Fatalf("expected CharacterId=%d, got %d", characterId, cmd.CharacterId)
	}
	if cmd.Body.CardId != uint32(cardId) {
		t.Fatalf("expected CardId=%d, got %d", uint32(cardId), cmd.Body.CardId)
	}
	// EventId is what atlas-monster-book dedupes card inserts on.
	if cmd.EventId != txnId {
		t.Fatalf("expected EventId=%s, got %s", txnId, cmd.EventId)
	}
	if cmd.Body.Source != mbmsg.SourceItemUse {
		t.Fatalf("expected Source=%q, got %q", mbmsg.SourceItemUse, cmd.Body.Source)
	}
}

// The item-use path and the drop-pickup path must produce the same command
// apart from the source marker, so atlas-monster-book handles both identically.
func TestMonsterBookCardPickedUpCommandProviderShape(t *testing.T) {
	tid := uuid.New()
	txnId := uuid.New()
	msgs, err := MonsterBookCardPickedUpCommandProvider(tid, 12, txnId, 2380006)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd mbmsg.Command[mbmsg.CardPickedUpBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Body.Source != mbmsg.SourceItemUse {
		t.Fatalf("expected Source=%q, got %q", mbmsg.SourceItemUse, cmd.Body.Source)
	}
	if cmd.Body.CardId != 2380006 {
		t.Fatalf("expected CardId=2380006, got %d", cmd.Body.CardId)
	}
	// Keyed by characterId so a character's registrations stay ordered.
	if string(msgs[0].Key) != string(kafkaProducer.CreateKey(12)) {
		t.Fatalf("expected key for character 12, got %q", string(msgs[0].Key))
	}
}

// Monster cards must reach ConsumeMonsterCard, not the ConsumeBare fallback.
func TestMonsterCardIsNotAStandardConsumer(t *testing.T) {
	if usesStandardConsumer(2380003) {
		t.Fatal("monster cards must not route through ConsumeStandard")
	}
	if item2.GetClassification(2380003) != item2.ClassificationConsumableMonsterCard {
		t.Fatalf("2380003 classification = %d, want %d", item2.GetClassification(2380003), item2.ClassificationConsumableMonsterCard)
	}
}
