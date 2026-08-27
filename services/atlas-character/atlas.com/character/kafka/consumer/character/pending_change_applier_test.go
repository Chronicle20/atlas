package character

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"atlas-character/pending_change"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newApplierTestDB starts a disposable Postgres container and migrates every
// table the applier touches: pending_change (with its partial unique
// indexes), characters (the apply path's write target), and the outbox every
// emission lands in. Mirrors pending_change/refund_idempotency_test.go's
// newProcessorTestDB — a fresh container per test rather than a shared helper,
// since the two packages' test files cannot see each other's unexported
// symbols.
func newApplierTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	for _, m := range []func(*gorm.DB) error{pending_change.Migration, character.Migration, outbox.Migration} {
		if err := m(db); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}
	return db
}

func applierTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

var applierTestTenantModel = func() tenant.Model {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		panic(err)
	}
	return tm
}()

func applierTestContext(t *testing.T) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), applierTestTenantModel)
}

// seedApplierCharacter creates a real character row through the character
// processor so the applier's own GetById / CheckNameValidity calls see exactly
// what production sees.
func seedApplierCharacter(t *testing.T, db *gorm.DB, name string, worldId world.Id) uint32 {
	t.Helper()
	l, ctx := applierTestLogger(t), applierTestContext(t)
	input := character.NewEmptyBuilder().
		SetAccountId(1000).
		SetWorldId(worldId).
		SetName(name).
		SetLevel(1).
		SetExperience(0).
		Build()
	c, err := character.NewProcessor(l, ctx, db).
		Create(message.NewBuffer())(uuid.New(), input, 0)
	if err != nil {
		t.Fatalf("seed character %s: %v", name, err)
	}
	return c.Id()
}

// countApplierOutboxMessagesMatching counts outbox rows whose serialized body
// contains substr. Reading the outbox directly is the point: it is the only
// place an emission survives the transaction, so counting here proves what
// actually reaches Kafka, across every invocation in the test.
func countApplierOutboxMessagesMatching(t *testing.T, db *gorm.DB, substr string) int {
	t.Helper()
	var n int64
	err := db.Model(&outbox.Entity{}).
		Where("encode(message_value, 'escape') LIKE ?", "%"+substr+"%").
		Count(&n).Error
	if err != nil {
		t.Fatalf("count outbox messages matching %q: %v", substr, err)
	}
	return int(n)
}

// FR-2.4: application must never mutate a character that is live in a channel.
// The LOGOUT event is the proof of absence; the applier is the only writer.
func TestLogoutAppliesAPendingNameChange(t *testing.T) {
	db := newApplierTestDB(t)
	l, ctx := applierTestLogger(t), applierTestContext(t)
	characterId := seedApplierCharacter(t, db, "Kilo", world.Id(0))

	pcp := pending_change.NewProcessor(l, ctx, db)
	m, err := pcp.CreateAndEmit(uuid.New(), characterId, pending_change.TypeNameChange, "Lima", world.Id(0), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	handleLogoutApplyPendingChanges(db)(l, ctx, character2.StatusEvent[character2.StatusEventLogoutBody]{
		TransactionId: uuid.New(), CharacterId: characterId, WorldId: world.Id(0),
		Type: character2.StatusEventTypeLogout,
	})

	c, err := character.NewProcessor(l, ctx, db).GetById()(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if c.Name() != "Lima" {
		t.Fatalf("name = %s, want Lima", c.Name())
	}
	got, err := pcp.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != pending_change.StatusApplied {
		t.Fatalf("status = %s, want APPLIED", got.Status())
	}
	if countApplierOutboxMessagesMatching(t, db, "NAME_CHANGED") != 1 {
		t.Fatal("expected exactly one NAME_CHANGED emission")
	}
}

// FR-2.7 / §5.2: the name was taken between reservation and apply.
func TestLogoutRejectsAndRefundsWhenTheNameWasTaken(t *testing.T) {
	db := newApplierTestDB(t)
	l, ctx := applierTestLogger(t), applierTestContext(t)
	characterId := seedApplierCharacter(t, db, "Mike", world.Id(0))
	assetId := uint32(9002)
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, err := pcp.CreateAndEmit(uuid.New(), characterId, pending_change.TypeNameChange, "November", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	// Another character takes the name in the interim.
	seedApplierCharacter(t, db, "November", world.Id(0))

	handleLogoutApplyPendingChanges(db)(l, ctx, character2.StatusEvent[character2.StatusEventLogoutBody]{
		TransactionId: uuid.New(), CharacterId: characterId, WorldId: world.Id(0),
		Type: character2.StatusEventTypeLogout,
	})

	got, err := pcp.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != pending_change.StatusRejected || got.Reason() != "name_taken" {
		t.Fatalf("got %s / %s, want REJECTED / name_taken", got.Status(), got.Reason())
	}
	if countApplierOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("expected the coupon to be refunded exactly once")
	}
}

// FR-2.9: a resolution that happened while the player was offline is delivered
// on their next login, not discarded.
func TestLoginReemitsAnUnnotifiedResolution(t *testing.T) {
	db := newApplierTestDB(t)
	l, ctx := applierTestLogger(t), applierTestContext(t)
	characterId := seedApplierCharacter(t, db, "Oscar", world.Id(0))
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, err := pcp.CreateAndEmit(uuid.New(), characterId, pending_change.TypeNameChange, "Papa", world.Id(0), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, _, err := pcp.ResolveAndEmit(m.Id(), pending_change.StatusCancelled, "operator_cancelled"); err != nil {
		t.Fatalf("ResolveAndEmit: %v", err)
	}

	before := countApplierOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED")
	handleLoginPendingChangeCatchUp(db)(l, ctx, character2.StatusEvent[character2.StatusEventLoginBody]{
		TransactionId: uuid.New(), CharacterId: characterId, WorldId: world.Id(0),
		Type: character2.StatusEventTypeLogin,
	})
	if got := countApplierOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED") - before; got != 1 {
		t.Fatalf("expected 1 re-emission, got %d", got)
	}
}

// Controller ruling carried from task-9: markNotified is the one emission in
// pending_change not previously gated on RowsAffected. Two LOGIN deliveries
// for the same character can race getResolvedUnnotified (a plain SELECT
// outside any transaction) and both observe the row unnotified; without the
// RowsAffected gate both would emit. The count spans BOTH deliveries
// together — counting only the second delivery's output would not catch a
// duplicate minted by the first and second together.
func TestConcurrentLoginDeliveriesReemitExactlyOnce(t *testing.T) {
	db := newApplierTestDB(t)
	l, ctx := applierTestLogger(t), applierTestContext(t)
	characterId := seedApplierCharacter(t, db, "Quebec", world.Id(0))
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, err := pcp.CreateAndEmit(uuid.New(), characterId, pending_change.TypeNameChange, "Romeo", world.Id(0), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, _, err := pcp.ResolveAndEmit(m.Id(), pending_change.StatusCancelled, "operator_cancelled"); err != nil {
		t.Fatalf("ResolveAndEmit: %v", err)
	}

	before := countApplierOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED")

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-start
			handleLoginPendingChangeCatchUp(db)(l, ctx, character2.StatusEvent[character2.StatusEventLoginBody]{
				TransactionId: uuid.New(), CharacterId: characterId, WorldId: world.Id(0),
				Type: character2.StatusEventTypeLogin,
			})
		}()
	}
	close(start)
	wg.Wait()

	if got := countApplierOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED") - before; got != 1 {
		t.Fatalf("expected exactly 1 notification across 2 concurrent LOGIN deliveries, got %d", got)
	}
}
