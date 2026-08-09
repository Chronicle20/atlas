package coupon

// NOTE ON THE HARNESS AND WHAT THESE TESTS DO / DO NOT PROVE.
//
// These tests run against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres. A human ruling on this
// branch selected SQLite in-memory as the harness for this plan's DB tests
// (testcontainers Postgres was available and deliberately declined).
//
// Each test pins an OUTCOME — which client-facing key a given seeded state
// produces, what the success path writes, and which producer path each event
// travels. NONE of them demonstrates a RACE: SQLite in-memory is capped to a
// single connection and serializes writers, so nothing here runs two
// simultaneous redemptions. The single-statement reservation and the unique
// index are what make the concurrent cases safe under Postgres; a passing run
// of this file is not evidence of that.
//
// The character and commodity lookups are REMOTE (HTTP) in production, so the
// processor's chaP and gf seams are overridden here with in-process stubs.
// That means these tests exercise the ladder, the transaction and the event
// routing — not the REST clients.

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/character"
	"atlas-cashshop/configuration"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/coupon/reward"
	kafkacashshop "atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/wallet"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	testAccountId   = uint32(9001)
	testCharacterId = uint32(4242)
	// testStatusTopic is what EVENT_TOPIC_CASH_SHOP_STATUS resolves to for the
	// duration of these tests. Both the direct producer and the outbox resolve
	// the token through topic.EnvProvider, so it has to be set either way.
	testStatusTopic = "test-cash-shop-status"
)

// processorTestTenant is shared by every test in this file ON PURPOSE.
// configuration.GetTenantConfig caches per tenant id and falls back to the
// documented defaults after a failed fetch; a fresh tenant per test would pay
// that failed fetch (and its retry budget) once per test.
var (
	processorTenantOnce sync.Once
	processorTenant     tenant.Model
	processorTenantErr  error
)

func processorTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	processorTenantOnce.Do(func() {
		processorTenant, processorTenantErr = tenant.Create(uuid.New(), "GMS", 83, 1)
	})
	require.NoError(t, processorTenantErr)
	return processorTenant
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// newProcessorTestEnv migrates every table one redemption touches, points the
// limiter at a fresh in-memory redis, and installs a capturing Kafka writer so
// the DIRECT producer path can be inspected (and so a failure emit does not
// spend ~42s retrying against an absent broker).
func newProcessorTestEnv(t *testing.T) (*gorm.DB, tenant.Model, context.Context, *directEvents) {
	t.Helper()
	t.Setenv("EVENT_TOPIC_CASH_SHOP_STATUS", testStatusTopic)

	db := databasetest.NewInMemoryTenantDB(t,
		Migration,
		redemption.Migration,
		wallet.Migration,
		sqliteCompartmentMigration,
		asset.Migration,
		outbox.Migration,
	)
	tm := processorTestTenant(t)

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	useLimiterStore(t, client)

	return db, tm, tenant.WithContext(context.Background(), tm), captureDirectEvents(t)
}

// --- direct-producer capture -------------------------------------------------

type directEvents struct {
	mu   sync.Mutex
	msgs []kafka.Message
}

type capturingWriter struct {
	topicName string
	sink      *directEvents
}

func (w capturingWriter) Topic() string { return w.topicName }

func (w capturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.sink.mu.Lock()
	defer w.sink.mu.Unlock()
	w.sink.msgs = append(w.sink.msgs, msgs...)
	return nil
}

func (w capturingWriter) Close() error { return nil }

// captureDirectEvents swaps the process-wide producer manager for one whose
// writers record instead of publishing.
func captureDirectEvents(t *testing.T) *directEvents {
	t.Helper()
	d := &directEvents{}
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(func(topicName string) producer.Writer {
		return capturingWriter{topicName: topicName, sink: d}
	}))
	t.Cleanup(producer.ResetInstance)
	return d
}

// lastCouponFailure returns the error key of the most recent COUPON_FAILED
// event seen on the direct path, or "" when none was emitted.
func (d *directEvents) lastCouponFailure() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := len(d.msgs) - 1; i >= 0; i-- {
		var e kafkacashshop.StatusEvent[kafkacashshop.CouponFailedBody]
		if err := json.Unmarshal(d.msgs[i].Value, &e); err != nil {
			continue
		}
		if e.Type == kafkacashshop.StatusEventTypeCouponFailed {
			return e.Body.Error
		}
	}
	return ""
}

func (d *directEvents) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.msgs)
}

// --- coupon-table query counter ---------------------------------------------

type queryCounter struct {
	mu      sync.Mutex
	coupons int
}

func (q *queryCounter) couponSelects() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.coupons
}

// countQueriesFrom counts SELECTs issued against the coupons table on db (and
// on every transaction derived from it). TestRedeemSuccessGrantsAndEmits
// asserts this counter is non-zero on a path that DOES read the table, which
// is what makes the zero-assertion in TestRedeemRateLimitedShortCircuits a
// real assertion rather than a counter that never fires.
func countQueriesFrom(t *testing.T, db *gorm.DB) *queryCounter {
	t.Helper()
	qc := &queryCounter{}
	const name = "coupontest:count_coupon_selects"
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(name, func(d *gorm.DB) {
		if d.Statement == nil {
			return
		}
		table := d.Statement.Table
		if table == "" && d.Statement.Schema != nil {
			table = d.Statement.Schema.Table
		}
		if table == "coupons" {
			qc.mu.Lock()
			qc.coupons++
			qc.mu.Unlock()
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(name) })
	return qc
}

// --- processor construction --------------------------------------------------

// stubCharacterProcessor stands in for the remote atlas-character lookup.
type stubCharacterProcessor struct {
	m   character.Model
	err error
}

func (s stubCharacterProcessor) GetById(_ ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	return func(_ uint32) (character.Model, error) { return s.m, s.err }
}

func (s stubCharacterProcessor) InventoryDecorator(m character.Model) character.Model { return m }

// testCharacter is an Explorer (job type 0), which resolves to
// compartment.TypeExplorer — the type seedCompartment writes.
func testCharacter(t *testing.T) character.Model {
	t.Helper()
	return character.NewModelBuilder().
		SetId(testCharacterId).
		SetAccountId(testAccountId).
		SetJobId(job.Id(100)).
		Build()
}

// newTestProcessor builds the real processor and replaces its two REMOTE
// seams: the character lookup and the granter factory (whose cash-item arm
// resolves a commodity over HTTP).
func newTestProcessor(t *testing.T, ctx context.Context, db *gorm.DB) Processor {
	t.Helper()
	p := NewProcessor(testLogger(t), ctx, db).(*ProcessorImpl)
	p.chaP = stubCharacterProcessor{m: testCharacter(t)}
	p.gf = func(l logrus.FieldLogger, gctx context.Context, r reward.Reward) (rewardGranter, error) {
		if r.Type() == reward.TypeCashItem {
			return cashItemGranter{l: l, ctx: gctx, cp: stubCommodityProcessor(t, couponRewardTemplateId)}, nil
		}
		return granterFor(l, gctx, r)
	}
	return p
}

// --- seeding / assertion helpers --------------------------------------------

func seedRedemption(t *testing.T, db *gorm.DB, tm tenant.Model, couponId uuid.UUID, accountId uint32) {
	t.Helper()
	rm, err := redemption.NewBuilder(couponId, accountId, testCharacterId).
		SetTransactionId(uuid.New()).
		SetRewardsGranted(reward.Rewards{reward.NewCurrencyReward(1, 1)}).
		SetRedeemedAt(time.Now()).
		Build()
	require.NoError(t, err)
	_, err = redemption.Create(db, tm, rm)
	require.NoError(t, err)
}

func countRedemptions(t *testing.T, db *gorm.DB, tm tenant.Model) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&redemption.Entity{}).Where("tenant_id = ?", tm.Id()).Count(&n).Error)
	return n
}

func loadOnlyRedemption(t *testing.T, db *gorm.DB, tm tenant.Model) redemption.Model {
	t.Helper()
	var rows []redemption.Entity
	require.NoError(t, db.Where("tenant_id = ?", tm.Id()).Find(&rows).Error)
	require.Len(t, rows, 1)
	m, err := redemption.Make(rows[0])
	require.NoError(t, err)
	return m
}

func countAssets(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&asset.Entity{}).Count(&n).Error)
	return n
}

// outboxCouponEvent is the decoded success event as it was enqueued.
type outboxCouponEvent struct {
	Body kafkacashshop.CouponRedeemedBody
}

// loadOnlyOutboxCouponEvent returns the single COUPON_REDEEMED event enqueued
// on the cash-shop status topic. The transaction also enqueues the wallet and
// asset events those processors emit, so the rows are filtered rather than
// counted.
func loadOnlyOutboxCouponEvent(t *testing.T, db *gorm.DB) outboxCouponEvent {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", testStatusTopic).Find(&rows).Error)

	var found []outboxCouponEvent
	for _, r := range rows {
		var e kafkacashshop.StatusEvent[kafkacashshop.CouponRedeemedBody]
		if err := json.Unmarshal(r.MessageValue, &e); err != nil {
			continue
		}
		if e.Type == kafkacashshop.StatusEventTypeCouponRedeemed {
			found = append(found, outboxCouponEvent{Body: e.Body})
		}
	}
	require.Len(t, found, 1, "expected exactly one COUPON_REDEEMED outbox row")
	return found[0]
}

func recordFailures(t *testing.T, ctx context.Context, tm tenant.Model, accountId uint32, n int) {
	t.Helper()
	l := NewLimiter(uint32(n)+1, time.Minute)
	for i := 0; i < n; i++ {
		require.NoError(t, l.RecordFailure(ctx, tm, accountId))
	}
}

// exhaustLimiter records enough failures to exceed the tenant's budget. The
// tenant has no configured limit, so the documented default applies.
func exhaustLimiter(t *testing.T, ctx context.Context, tm tenant.Model, accountId uint32) {
	t.Helper()
	recordFailures(t, ctx, tm, accountId, configuration.DefaultCouponAttempts)
}

func limiterCount(t *testing.T, ctx context.Context, tm tenant.Model, accountId uint32) int64 {
	t.Helper()
	n, err := limiterStore.Get(ctx, tm, limiterKey(accountId))
	require.NoError(t, err)
	return n
}

// --- tests -------------------------------------------------------------------

func TestRedeemLadderOutcomes(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		name    string
		seed    func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string
		wantKey string
	}{
		{
			"no such code",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string { return "NOSUCHCODE" },
			ErrorKeyInvalidCode,
		},
		{
			"inactive code",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				seedCoupon(t, db, tm, NewBuilder("OFF").SetActive(false).SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
				return "OFF"
			},
			ErrorKeyNotRegistered,
		},
		{
			"not started yet",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				seedCoupon(t, db, tm, NewBuilder("EARLY").SetStartsAt(ptrTime(now.Add(time.Hour))).SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
				return "EARLY"
			},
			ErrorKeyNotRegistered,
		},
		{
			"expired",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				seedCoupon(t, db, tm, NewBuilder("OLD").SetExpiresAt(ptrTime(now.Add(-time.Hour))).SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
				return "OLD"
			},
			ErrorKeyExpired,
		},
		{
			"already redeemed by this account",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				id := seedCoupon(t, db, tm, NewBuilder("TWICE").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
				seedRedemption(t, db, tm, id, testAccountId)
				return "TWICE"
			},
			ErrorKeyAlreadyUsed,
		},
		{
			"global uses exhausted",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				seedCoupon(t, db, tm, NewBuilder("GONE").SetMaxUses(ptrU32(1)).SetRedemptionCount(1).SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
				return "GONE"
			},
			ErrorKeyUsageLimit,
		},
		{
			"locker has no room for the item reward",
			func(t *testing.T, db *gorm.DB, tm tenant.Model, ctx context.Context) string {
				seedFullCompartment(t, db, ctx, testAccountId)
				seedCoupon(t, db, tm, NewBuilder("ITEM").SetRewards(reward.Rewards{reward.NewCashItemReward(50200000, 1)}))
				return "ITEM"
			},
			ErrorKeyInventoryFull,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			db, tm, ctx, events := newProcessorTestEnv(t)
			seedWallet(t, db, ctx, testAccountId, 100, 200, 300)
			code := c.seed(t, db, tm, ctx)

			err := newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, code)
			if err == nil {
				t.Fatal("want a rejection")
			}
			if got := events.lastCouponFailure(); got != c.wantKey {
				t.Errorf("emitted key = %q, want %q", got, c.wantKey)
			}
			// The "already redeemed" case seeds one row; every other case must
			// leave the table exactly as the seed left it.
			want := int64(0)
			if c.name == "already redeemed by this account" {
				want = 1
			}
			if got := countRedemptions(t, db, tm); got != want {
				t.Errorf("redemption rows = %d, want %d", got, want)
			}
			if p := loadWalletPoints(t, db, ctx, testAccountId); p != 200 {
				t.Errorf("wallet points = %d, want the seeded 200 (unchanged)", p)
			}
			// A rejection asserts "nothing happened"; it must not ride the
			// outbox, which implies a commit.
			var outboxRows int64
			require.NoError(t, db.Model(&outbox.Entity{}).Count(&outboxRows).Error)
			if outboxRows != 0 {
				t.Errorf("outbox rows = %d, want 0 for a rejected redemption", outboxRows)
			}
		})
	}
}

func TestRedeemSuccessGrantsAndEmits(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedEmptyCompartment(t, db, ctx, testAccountId)
	seedCoupon(t, db, tm, NewBuilder("WIN").SetRewards(reward.Rewards{
		reward.NewCurrencyReward(2, 1500),
		reward.NewCashItemReward(50200000, 1),
	}))
	queries := countQueriesFrom(t, db)

	// A lowercase, padded submission must match the stored (normalized) code.
	if err := newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "  win  "); err != nil {
		t.Fatalf("RedeemAndEmit: %v", err)
	}

	// Positive control for the counter the rate-limit test asserts is zero.
	if queries.couponSelects() == 0 {
		t.Error("coupon lookups = 0 on a path that reads the coupons table; the counter is not wired")
	}

	r := loadOnlyRedemption(t, db, tm)
	if r.AccountId() != testAccountId {
		t.Errorf("redemption accountId = %d, want %d", r.AccountId(), testAccountId)
	}
	if len(r.RewardsGranted()) != 2 {
		t.Errorf("rewardsGranted = %d entries, want 2 (a snapshot of what was granted)", len(r.RewardsGranted()))
	}
	if p := loadWalletPoints(t, db, ctx, testAccountId); p != 1500 {
		t.Errorf("points = %d, want 1500", p)
	}
	if n := countAssets(t, db); n != 1 {
		t.Errorf("locker assets = %d, want 1", n)
	}
	// The success event rides the OUTBOX, not the direct path.
	e := loadOnlyOutboxCouponEvent(t, db)
	if e.Body.MaplePoints != 1500 {
		t.Errorf("maplePoints = %d, want the 1500 DELTA this coupon awarded", e.Body.MaplePoints)
	}
	if len(e.Body.AssetIds) != 1 {
		t.Errorf("assetIds = %v, want one", e.Body.AssetIds)
	}
	if n := events.count(); n != 0 {
		t.Errorf("direct-path messages = %d, want 0; the success event belongs to the outbox", n)
	}
}

// TestRedeemPrepaidOnlyCouponStillSucceeds pins the all-zero grantedReward
// trap: wallet.Award routes every currency other than 1 (credit) and 2 (maple
// points) to PREPAID, and currencyGranter reports neither field for those, so
// a prepaid coupon mutates the wallet and returns a ZERO grantedReward. The
// success path is gated on err == nil, never on a non-zero aggregate — a
// zero-check here would silently swallow the whole reward.
func TestRedeemPrepaidOnlyCouponStillSucceeds(t *testing.T) {
	db, tm, ctx, _ := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("PREPAID").SetRewards(reward.Rewards{reward.NewCurrencyReward(3, 750)}))

	if err := newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "PREPAID"); err != nil {
		t.Fatalf("RedeemAndEmit: %v", err)
	}
	if got := countRedemptions(t, db, tm); got != 1 {
		t.Errorf("redemption rows = %d, want 1", got)
	}
	var w wallet.Entity
	require.NoError(t, db.WithContext(ctx).Where("account_id = ?", testAccountId).First(&w).Error)
	if w.Prepaid != 750 {
		t.Errorf("prepaid = %d, want 750", w.Prepaid)
	}
	// COUPON_REDEEMED must still be emitted even though every field of the
	// aggregated grant is zero.
	e := loadOnlyOutboxCouponEvent(t, db)
	if e.Body.MaplePoints != 0 || e.Body.Credit != 0 || len(e.Body.AssetIds) != 0 {
		t.Errorf("event body = %+v, want an all-zero body (prepaid has no field)", e.Body)
	}
}

func TestRedeemRateLimitedShortCircuits(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedCoupon(t, db, tm, NewBuilder("REAL").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
	exhaustLimiter(t, ctx, tm, testAccountId)
	queries := countQueriesFrom(t, db)

	_ = newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "REAL")

	// The whole point of the limiter is that a blocked attempt does NOT hit
	// the coupons table.
	if queries.couponSelects() != 0 {
		t.Errorf("coupon lookups = %d, want 0 for a rate-limited attempt", queries.couponSelects())
	}
	// It reports INVALID_COUPON_CODE, never a distinct "rate limited" key,
	// so a blocked attacker cannot tell a real code from a fake one.
	if got := events.lastCouponFailure(); got != ErrorKeyInvalidCode {
		t.Errorf("emitted key = %q, want %q", got, ErrorKeyInvalidCode)
	}
}

func TestRedeemSuccessResetsTheLimiter(t *testing.T) {
	db, tm, ctx, _ := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("OK").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
	recordFailures(t, ctx, tm, testAccountId, 3)

	if err := newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "OK"); err != nil {
		t.Fatal(err)
	}
	if n := limiterCount(t, ctx, tm, testAccountId); n != 0 {
		t.Errorf("limiter count = %d after a success, want 0", n)
	}
}

func TestRedeemImplausibleCodeIsRejectedWithoutAQuery(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedCoupon(t, db, tm, NewBuilder("REAL").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
	queries := countQueriesFrom(t, db)

	// 33 characters — one over the column limit, so it cannot name any row.
	err := newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "123456789012345678901234567890123")
	if err == nil {
		t.Fatal("want a rejection")
	}
	if queries.couponSelects() != 0 {
		t.Errorf("coupon lookups = %d, want 0 for an implausible code", queries.couponSelects())
	}
	if got := events.lastCouponFailure(); got != ErrorKeyInvalidCode {
		t.Errorf("emitted key = %q, want %q", got, ErrorKeyInvalidCode)
	}
}

func TestRedeemFailureCountsAgainstTheLimiter(t *testing.T) {
	db, tm, ctx, _ := newProcessorTestEnv(t)
	seedCoupon(t, db, tm, NewBuilder("REAL").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))

	_ = newTestProcessor(t, ctx, db).RedeemAndEmit(testCharacterId, "WRONG")

	if n := limiterCount(t, ctx, tm, testAccountId); n != 1 {
		t.Errorf("limiter count = %d after one failed attempt, want 1", n)
	}
}

func TestRedeemUnresolvableCharacterReportsUnknown(t *testing.T) {
	db, _, ctx, events := newProcessorTestEnv(t)
	p := NewProcessor(testLogger(t), ctx, db).(*ProcessorImpl)
	p.chaP = stubCharacterProcessor{err: gorm.ErrRecordNotFound}

	if err := p.RedeemAndEmit(testCharacterId, "ANY"); err == nil {
		t.Fatal("want a rejection")
	}
	if got := events.lastCouponFailure(); got != ErrorKeyUnknown {
		t.Errorf("emitted key = %q, want %q", got, ErrorKeyUnknown)
	}
}

func TestCrudRoundTrip(t *testing.T) {
	db, tm, ctx, _ := newProcessorTestEnv(t)
	p := newTestProcessor(t, ctx, db)

	m, err := NewBuilder("crud").SetDescription("first").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 5)}).Build()
	require.NoError(t, err)
	created, err := p.Create(m)
	require.NoError(t, err)
	if created.Code() != "CRUD" {
		t.Errorf("stored code = %q, want the normalized %q", created.Code(), "CRUD")
	}

	got, err := p.GetById(created.Id())
	require.NoError(t, err)
	if got.Description() != "first" {
		t.Errorf("description = %q, want %q", got.Description(), "first")
	}

	byCode, err := p.GetByCode("  crud ")
	require.NoError(t, err)
	if byCode.Id() != created.Id() {
		t.Error("GetByCode did not normalize its argument")
	}

	edit, err := NewBuilder("CRUD").SetDescription("second").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 5)}).Build()
	require.NoError(t, err)
	updated, err := p.Update(created.Id(), edit)
	require.NoError(t, err)
	if updated.Description() != "second" {
		t.Errorf("description = %q, want %q", updated.Description(), "second")
	}

	all, err := p.GetAll(Filters{})
	require.NoError(t, err)
	if len(all) != 1 {
		t.Errorf("GetAll = %d coupons, want 1", len(all))
	}

	require.NoError(t, p.Delete(created.Id()))
	if _, err = p.GetById(created.Id()); err == nil {
		t.Error("GetById succeeded after Delete")
	}

	// A redeemed coupon is refused: deleting it would orphan its audit trail.
	id := seedCoupon(t, db, tm, NewBuilder("REDEEMED").SetRewards(reward.Rewards{reward.NewCurrencyReward(1, 1)}))
	seedRedemption(t, db, tm, id, testAccountId)
	if err = p.Delete(id); err == nil {
		t.Error("Delete of a redeemed coupon succeeded, want ErrHasRedemptions")
	}
}
