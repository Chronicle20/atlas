package shops

import (
	"atlas-npc/commodities"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// enterTestDB mirrors confirmTestDB (processor_confirm_test.go): a fresh
// per-test in-memory SQLite database migrated for both the shop and
// commodity tables. Importing atlas-npc/test here is not an option — its
// fixtures.go imports this package, which would be an import cycle.
func enterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err = commodities.Migration(db); err != nil {
		t.Fatalf("failed to migrate commodities: %v", err)
	}
	if err = Migration(db); err != nil {
		t.Fatalf("failed to migrate shops: %v", err)
	}
	return db
}

func enterTestContext(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func enterTestProcessor(t *testing.T, ctx context.Context, db *gorm.DB) *ProcessorImpl {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	// GetByNpcId always runs the rechargeable-consumables decorator, which
	// hits the package-level consumable cache singleton; short-circuit it,
	// mirroring TestShopsProcessor in processor_test.go.
	p.RechargeableConsumablesDecoratorFn = func(m Model) Model {
		return m
	}
	return p
}

// TestEnter_ShopNotFound_EmitsEnterError: without this the saga step hangs
// until the saga timer expires and the player's client never unlocks
// (task-221 design delta D3).
func TestEnter_ShopNotFound_EmitsEnterError(t *testing.T) {
	setupTestRegistry(t)
	ctx := enterTestContext(t)
	db := enterTestDB(t)
	p := enterTestProcessor(t, ctx, db)

	mb := message.NewBuffer()
	txn := uuid.New()

	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter returned err %v; the failure must be reported on the topic, not returned", err)
	}

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnterErrorBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != shops.StatusEventTypeEnterError {
		t.Errorf("Type = %q, want %q", e.Type, shops.StatusEventTypeEnterError)
	}
	if e.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", e.TransactionId, txn)
	}
	if e.Body.Reason != shops.EnterErrorShopNotFound {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, shops.EnterErrorShopNotFound)
	}
}

// TestEnter_AlreadyInShop_EmitsEnterError: AddCharacter used to overwrite
// unconditionally, so a second ENTER silently re-entered and a remote-merchant
// saga would consume the item for a shop the player was already standing in
// (task-221 design delta D4, PRD FR-2.3).
func TestEnter_AlreadyInShop_EmitsEnterError(t *testing.T) {
	setupTestRegistry(t)
	ctx := enterTestContext(t)
	db := enterTestDB(t)
	p := enterTestProcessor(t, ctx, db)

	if _, err := p.CreateShop(9090000, false, nil); err != nil {
		t.Fatalf("CreateShop: %v", err)
	}

	GetRegistry().AddCharacter(ctx, 1234, 9090000)
	t.Cleanup(func() { GetRegistry().RemoveCharacter(ctx, 1234) })

	mb := message.NewBuffer()
	txn := uuid.New()
	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnterErrorBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Body.Reason != shops.EnterErrorAlreadyInShop {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, shops.EnterErrorAlreadyInShop)
	}
}

// TestEnter_Success_EmitsEnteredWithTransactionId
func TestEnter_Success_EmitsEnteredWithTransactionId(t *testing.T) {
	setupTestRegistry(t)
	ctx := enterTestContext(t)
	db := enterTestDB(t)
	p := enterTestProcessor(t, ctx, db)

	if _, err := p.CreateShop(9090000, false, nil); err != nil {
		t.Fatalf("CreateShop: %v", err)
	}

	mb := message.NewBuffer()
	txn := uuid.New()
	if err := p.Enter(mb)(txn)(1234)(9090000); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	t.Cleanup(func() { GetRegistry().RemoveCharacter(ctx, 1234) })

	msgs := mb.GetAll()[shops.EnvStatusEventTopic]
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var e shops.StatusEvent[shops.StatusEventEnteredBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != shops.StatusEventTypeEntered || e.TransactionId != txn || e.Body.NpcTemplateId != 9090000 {
		t.Errorf("unexpected entered event: %+v", e)
	}
}
