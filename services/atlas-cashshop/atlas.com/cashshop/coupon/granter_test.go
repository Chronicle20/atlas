package coupon

// NOTE ON THE HARNESS AND WHAT THESE TESTS DO / DO NOT PROVE.
//
// These tests run against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres. A human ruling on this
// branch selected SQLite in-memory as the harness for this plan's DB tests
// (testcontainers Postgres was available and deliberately declined).
//
// TestCashItemGranterRechecksCapacityInsideTheTransaction pins an OUTCOME —
// that a full compartment yields a RedemptionError keyed INVENTORY_FULL from
// inside the granter, on the transaction handle it was given. It does NOT
// demonstrate the RACE it exists to close: nothing here runs two concurrent
// redemptions, and SQLite in-memory is capped to a single connection and
// serializes writers, so it could not. Read it as "the granter re-checks",
// not as "the TOCTOU window is proven closed".

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/wallet"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// couponRewardTemplateId is the item id the stub commodity resolves
// serialNumber 50200000 to. It is an arbitrary test value, not a claim about
// any real commodity row.
const couponRewardTemplateId = uint32(5220000)

// sqliteCompartment mirrors compartment.Entity's columns exactly, minus the
// Postgres-only `default:uuid_generate_v4()` on the primary key, which SQLite
// cannot parse ("near \"(\": syntax error" out of AutoMigrate). Seeding and
// the granter both go through compartment.Entity against the table this
// creates. The cost of the mirror is real: a future column added to
// compartment.Entity will not appear here until someone adds it, so these
// tests validate the granter's behaviour, not compartment.Migration itself.
type sqliteCompartment struct {
	Id        uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId  uuid.UUID `gorm:"not null"`
	AccountId uint32    `gorm:"not null"`
	Type      byte      `gorm:"not null"`
	Capacity  uint32    `gorm:"not null;default:55"`
}

func (sqliteCompartment) TableName() string { return "cash_compartments" }

func sqliteCompartmentMigration(db *gorm.DB) error { return db.AutoMigrate(&sqliteCompartment{}) }

// newGranterTestDB migrates the three tables a granter writes to: the wallet
// (accounts), the cash compartment, and the cash asset.
func newGranterTestDB(t *testing.T) (*gorm.DB, tenant.Model) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, wallet.Migration, sqliteCompartmentMigration, asset.Migration)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return db, tm
}

func testLoggerAndContext(t *testing.T, tm tenant.Model) (logrus.FieldLogger, context.Context) {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l, tenant.WithContext(context.Background(), tm)
}

func seedWallet(t *testing.T, db *gorm.DB, ctx context.Context, accountId uint32, credit uint32, points uint32, prepaid uint32) {
	t.Helper()
	e := &wallet.Entity{Id: uuid.New(), AccountId: accountId, Credit: credit, Points: points, Prepaid: prepaid}
	require.NoError(t, db.WithContext(ctx).Create(e).Error)
}

func loadWalletPoints(t *testing.T, db *gorm.DB, ctx context.Context, accountId uint32) uint32 {
	t.Helper()
	var e wallet.Entity
	require.NoError(t, db.WithContext(ctx).Where("account_id = ?", accountId).First(&e).Error)
	return e.Points
}

func seedCompartment(t *testing.T, db *gorm.DB, ctx context.Context, accountId uint32, capacity uint32) uuid.UUID {
	t.Helper()
	e := &compartment.Entity{Id: uuid.New(), AccountId: accountId, Type: byte(compartment.TypeExplorer), Capacity: capacity}
	require.NoError(t, db.WithContext(ctx).Create(e).Error)
	return e.Id
}

// seedFullCompartment builds a compartment whose capacity equals the number of
// assets already in it.
func seedFullCompartment(t *testing.T, db *gorm.DB, ctx context.Context, accountId uint32) uuid.UUID {
	t.Helper()
	id := seedCompartment(t, db, ctx, accountId, 1)
	e := &asset.Entity{CompartmentId: id, CashId: 12345, TemplateId: couponRewardTemplateId, Quantity: 1, PurchasedBy: accountId}
	require.NoError(t, db.WithContext(ctx).Create(e).Error)
	return id
}

func seedEmptyCompartment(t *testing.T, db *gorm.DB, ctx context.Context, accountId uint32) uuid.UUID {
	t.Helper()
	return seedCompartment(t, db, ctx, accountId, 55)
}

// stubCommodity stands in for the remote atlas-data commodity lookup. The real
// commodity.Processor issues an HTTP request, which a unit test has no service
// to answer.
type stubCommodity struct {
	m   commodity.Model
	err error
}

func (s stubCommodity) GetById(_ uint32) (commodity.Model, error) { return s.m, s.err }

func stubCommodityProcessor(t *testing.T, itemId uint32) commodity.Processor {
	t.Helper()
	m, err := commodity.Extract(commodity.RestModel{Id: 50200000, ItemId: itemId, Count: 1, Period: 30})
	require.NoError(t, err)
	return stubCommodity{m: m}
}

func TestGranterForDispatchesByType(t *testing.T) {
	_, tm := newGranterTestDB(t)
	l, ctx := testLoggerAndContext(t, tm)
	if _, err := granterFor(l, ctx, NewCurrencyReward(1, 5)); err != nil {
		t.Errorf("currency: %v", err)
	}
	if _, err := granterFor(l, ctx, NewCashItemReward(50200000, 1)); err != nil {
		t.Errorf("cash item: %v", err)
	}
	// An unknown type must be a hard error, not a silent no-op: a coupon that
	// claims to grant something and grants nothing is worse than one that fails.
	if _, err := granterFor(l, ctx, Reward{rewardType: "MESO", amount: 1}); err == nil {
		t.Error("want an error for an unknown reward type")
	}
}

func TestCurrencyGranterCreditsTheWalletAndReportsTheDelta(t *testing.T) {
	db, tm := newGranterTestDB(t)
	l, gctx := testLoggerAndContext(t, tm)
	seedWallet(t, db, gctx, 1001, 100, 200, 300) // credit, points, prepaid
	g, err := granterFor(l, gctx, NewCurrencyReward(2, 500))
	require.NoError(t, err)
	mb := message.NewBuffer()

	got, err := g.Grant(mb)(db, redemptionContext{accountId: 1001}, NewCurrencyReward(2, 500))
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	// The event body carries the DELTA the coupon awarded, not the balance.
	if got.maplePoints != 500 {
		t.Errorf("maplePoints delta = %d, want 500", got.maplePoints)
	}
	if got.credit != 0 {
		t.Errorf("credit delta = %d, want 0", got.credit)
	}
	if p := loadWalletPoints(t, db, gctx, 1001); p != 700 {
		t.Errorf("stored points = %d, want 700", p)
	}
}

func TestCashItemGranterRechecksCapacityInsideTheTransaction(t *testing.T) {
	// Q6: the ladder's pre-flight capacity check gives a deterministic error
	// ordering; THIS re-check is the granter's own. See the file header for
	// what this test does and does not demonstrate.
	db, tm := newGranterTestDB(t)
	l, gctx := testLoggerAndContext(t, tm)
	cid := seedFullCompartment(t, db, gctx, 1002) // capacity == len(assets)
	// granterFor's real commodity processor is used deliberately: the capacity
	// re-check must run BEFORE any commodity resolution, so a full locker
	// never depends on a remote lookup succeeding.
	g, err := granterFor(l, gctx, NewCashItemReward(50200000, 1))
	require.NoError(t, err)
	mb := message.NewBuffer()

	_, err = g.Grant(mb)(db, redemptionContext{accountId: 1002, compartmentId: cid}, NewCashItemReward(50200000, 1))
	var re *RedemptionError
	if !errors.As(err, &re) || re.Key() != ErrorKeyInventoryFull {
		t.Fatalf("err = %v, want a RedemptionError with key %s", err, ErrorKeyInventoryFull)
	}
}

func TestCashItemGranterReturnsTheAssetId(t *testing.T) {
	db, tm := newGranterTestDB(t)
	l, gctx := testLoggerAndContext(t, tm)
	cid := seedEmptyCompartment(t, db, gctx, 1003)
	g := cashItemGranter{l: l, ctx: gctx, cp: stubCommodityProcessor(t, couponRewardTemplateId)}
	mb := message.NewBuffer()

	got, err := g.Grant(mb)(db, redemptionContext{accountId: 1003, characterId: 77, compartmentId: cid}, NewCashItemReward(50200000, 1))
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if got.assetId == 0 {
		t.Error("assetId = 0; the channel needs it to build the CashInventoryItem")
	}
	// The locker row must name the ITEM, not the commodity serial: asset.Create
	// stores templateId verbatim and only consults commodityId for the period.
	var e asset.Entity
	require.NoError(t, db.WithContext(gctx).Where("id = ?", got.assetId).First(&e).Error)
	if e.TemplateId != couponRewardTemplateId {
		t.Errorf("stored templateId = %d, want %d", e.TemplateId, couponRewardTemplateId)
	}
	if e.CommodityId != 50200000 {
		t.Errorf("stored commodityId = %d, want 50200000", e.CommodityId)
	}
}
