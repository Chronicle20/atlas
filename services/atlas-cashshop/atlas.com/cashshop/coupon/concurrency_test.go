package coupon

// NOTE ON THE HARNESS AND WHAT THESE TESTS DO / DO NOT PROVE.
//
// These tests run against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres. A human ruling on this
// branch selected SQLite in-memory as the harness for this plan's DB tests
// (testcontainers Postgres was available and deliberately declined).
//
// What this file DOES demonstrate:
//   - the OUTCOME of two overlapping redemptions of a max_uses = 1 coupon:
//     exactly one success, exactly one COUPON_USAGE_LIMIT, redemption_count
//     left at 1, exactly one redemption row;
//   - the OUTCOME of two overlapping redemptions by the SAME account: exactly
//     one success, exactly one COUPON_ALREADY_USED, the reward granted once;
//   - that a failure in the reward loop rolls back EVERYTHING the redemption
//     had already written — the wallet credit, the reservation, the redemption
//     row — and leaves the code redeemable (the proof of design §2's
//     single-local-transaction decision, and the most valuable test here);
//   - that codes differing only in case or surrounding whitespace resolve to
//     the same coupon;
//   - freedom from data races in ATLAS'S OWN code under `-race`: the shared
//     *gorm.DB, the package-level limiter store, the prometheus vecs and the
//     capturing producer are all touched from two goroutines here, and that
//     part of the exercise is real.
//
// What this file does NOT demonstrate:
//   - true write concurrency. databasetest.NewInMemoryTenantDB caps the pool
//     at SetMaxOpenConns(1) (testdb.go:37), so the two goroutines below cannot
//     overlap inside the database; one transaction runs to completion before
//     the other acquires the connection. The tests pin the outcome, not the
//     interleaving.
//   - the Postgres unique-index path. redemption.IsUniqueViolation matches only
//     *pgconn.PgError 23505, which SQLite never produces. Under this harness the
//     same-account loser is decided by the ladder's prior-redemption Count, NOT
//     by idx_redemptions_tenant_coupon_account. See
//     TestConcurrentRedemptionsBySameAccount.
//   - reserveUse's RowsAffected verdict UNDER CONTENTION. Because writers
//     serialize, substituting a read-then-write for the conditional UPDATE does
//     not make these tests fail; that falsification requires Postgres.
//
// Do not read a passing run of this file as proof of race-safety.

import (
	"atlas-cashshop/character"
	"atlas-cashshop/coupon/reward"
	kafkacashshop "atlas-cashshop/kafka/message/cashshop"
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

const (
	// secondAccountId / secondCharacterId are a DIFFERENT account, so the
	// max-uses test is decided by the global reservation rather than by the
	// one-time-per-account rule.
	secondAccountId   = uint32(9002)
	secondCharacterId = uint32(4243)

	// petCommodityItemId resolves to item classification 500 (pet), which
	// cashItemGranter refuses outright: it creates the locker asset but not the
	// pet row behind it. That refusal is a REAL production guard
	// (granter.go:121-124), used here as the reward-loop failure — see
	// TestGrantFailureRollsEverythingBack for why filling the compartment
	// cannot serve.
	petCommodityItemId = uint32(5000000)
)

// couponFailureKeys returns the error key of every COUPON_FAILED event seen on
// the direct path, oldest first.
func (d *directEvents) couponFailureKeys() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []string
	for _, m := range d.msgs {
		var e kafkacashshop.StatusEvent[kafkacashshop.CouponFailedBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			continue
		}
		if e.Type == kafkacashshop.StatusEventTypeCouponFailed {
			out = append(out, e.Body.Error)
		}
	}
	return out
}

// countUpdatesTo counts UPDATE statements issued against the named table on db
// and on every transaction derived from it. It is the positive control for
// TestGrantFailureRollsEverythingBack: without it, "the wallet is unchanged"
// would hold just as well for a run in which the wallet was never written at
// all, and the test would prove nothing about rollback.
func countUpdatesTo(t *testing.T, db *gorm.DB, table string) func() int {
	t.Helper()
	var mu sync.Mutex
	n := 0
	name := "coupontest:count_updates_" + table
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(name, func(d *gorm.DB) {
		if d.Statement == nil {
			return
		}
		st := d.Statement.Table
		if st == "" && d.Statement.Schema != nil {
			st = d.Statement.Schema.Table
		}
		if st == table {
			mu.Lock()
			n++
			mu.Unlock()
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Update().Remove(name) })
	return func() int {
		mu.Lock()
		defer mu.Unlock()
		return n
	}
}

// characterModel builds the Explorer whose account owns the wallet and the
// compartment seeded for it. Explorer (job type 0) resolves to
// compartment.TypeExplorer, which is what seedCompartment writes.
func characterModel(t *testing.T, characterId uint32, accountId uint32) character.Model {
	t.Helper()
	return character.NewBuilder().
		SetId(characterId).
		SetAccountId(accountId).
		SetJobId(job.Id(100)).
		Build()
}

// newProcessorForCharacter is newTestProcessor with both remote seams
// parameterized: which character the session resolves to, and which item id the
// commodity lookup returns. Everything else is the real processor.
func newProcessorForCharacter(t *testing.T, ctx context.Context, db *gorm.DB, cm character.Model, commodityItemId uint32) Processor {
	t.Helper()
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	p.chaP = stubCharacterProcessor{m: cm}
	p.gf = func(gl logrus.FieldLogger, gctx context.Context, r reward.Reward) (rewardGranter, error) {
		if r.Type() == reward.TypeCashItem {
			return cashItemGranter{l: gl, ctx: gctx, cp: stubCommodityProcessor(t, commodityItemId)}, nil
		}
		return granterFor(gl, gctx, r)
	}
	return p
}

// runConcurrently launches one RedeemAndEmit per character on its own
// goroutine, released together, and returns their errors in order.
func runConcurrently(t *testing.T, ctx context.Context, db *gorm.DB, code string, characters ...character.Model) []error {
	t.Helper()
	errs := make([]error, len(characters))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, cm := range characters {
		wg.Add(1)
		go func(i int, cm character.Model) {
			defer wg.Done()
			p := newProcessorForCharacter(t, ctx, db, cm, couponRewardTemplateId)
			<-start
			errs[i] = p.RedeemAndEmit(cm.Id(), code)
		}(i, cm)
	}
	close(start)
	wg.Wait()
	return errs
}

func countSuccesses(errs []error) int {
	n := 0
	for _, e := range errs {
		if e == nil {
			n++
		}
	}
	return n
}

// TestConcurrentRedemptionsOfASingleUseCoupon drives two redemptions of a
// max_uses = 1 coupon from two goroutines, on behalf of two DIFFERENT accounts,
// so the outcome turns on the global reservation rather than on the
// one-time-per-account rule.
//
// Under this harness the two transactions do not actually overlap (see the
// file header), so this pins the OUTCOME — one winner, one COUPON_USAGE_LIMIT,
// a redemption_count of exactly 1 — and, under `-race`, the absence of data
// races in Atlas's own shared state. It is NOT evidence that reserveUse's
// conditional UPDATE is what produced that outcome; substituting a
// read-then-write does not fail this test here.
func TestConcurrentRedemptionsOfASingleUseCoupon(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedWallet(t, db, ctx, secondAccountId, 0, 0, 0)
	id := seedCoupon(t, db, tm, NewBuilder("ONESHOT").SetMaxUses(ptrU32(1)).
		SetRewards(reward.Rewards{reward.NewCurrencyReward(2, 100)}))

	errs := runConcurrently(t, ctx, db, "ONESHOT",
		characterModel(t, testCharacterId, testAccountId),
		characterModel(t, secondCharacterId, secondAccountId),
	)

	if got := countSuccesses(errs); got != 1 {
		t.Fatalf("successes = %d, want exactly 1 (errs = %v)", got, errs)
	}
	if got := events.couponFailureKeys(); len(got) != 1 || got[0] != ErrorKeyUsageLimit {
		t.Errorf("failure keys = %v, want exactly [%s]", got, ErrorKeyUsageLimit)
	}
	if n := loadCount(t, db, id); n != 1 {
		t.Errorf("redemptionCount = %d, want 1 — the loser must not have incremented it", n)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows = %d, want 1", n)
	}
}

// TestConcurrentRedemptionsBySameAccount drives two redemptions of the same
// code by the SAME account from two goroutines.
//
// IMPORTANT: under this harness the loser is decided by the ladder's
// redemption.CountByCouponAndAccount check (processor.go step 5), NOT by the
// unique index on (tenant_id, coupon_id, account_id). The writers serialize, so
// the second transaction begins after the first has committed and sees the
// prior row; and redemption.IsUniqueViolation matches only *pgconn.PgError
// 23505, which SQLite never raises. The index is what makes this safe under
// Postgres when the two transactions genuinely overlap — this test is not
// evidence of that path.
func TestConcurrentRedemptionsBySameAccount(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("SAMEACCT").SetRewards(reward.Rewards{reward.NewCurrencyReward(2, 100)}))

	cm := characterModel(t, testCharacterId, testAccountId)
	errs := runConcurrently(t, ctx, db, "SAMEACCT", cm, cm)

	if got := countSuccesses(errs); got != 1 {
		t.Fatalf("successes = %d, want exactly 1 (errs = %v)", got, errs)
	}
	if got := events.couponFailureKeys(); len(got) != 1 || got[0] != ErrorKeyAlreadyUsed {
		t.Errorf("failure keys = %v, want exactly [%s]", got, ErrorKeyAlreadyUsed)
	}
	if p := loadWalletPoints(t, db, ctx, testAccountId); p != 100 {
		t.Errorf("points = %d, want 100 — the reward must be granted exactly once", p)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows = %d, want 1", n)
	}
}

// TestGrantFailureRollsEverythingBack is the proof of design §2's
// local-transaction decision, and it is FULLY valid under this harness: no
// concurrency is involved. A failure in the reward loop must leave NOTHING
// behind — with a saga this would be a compensation assertion; with a single
// transaction it is a rollback assertion.
//
// The failure is the cash-item granter's pet refusal (granter.go:121-124), a
// real production guard, reached AFTER the currency reward has already credited
// the wallet and after reserveUse has already incremented redemption_count.
//
// Filling the compartment to capacity — the obvious choice — cannot serve here:
// the ladder's pre-flight capacity check (processor.go step 7) uses the same
// predicate as the granter's re-check, so a full locker is rejected BEFORE the
// reservation and before any grant. Nothing would have been written, so the
// assertions below would hold vacuously. That case is already covered by
// TestRedeemLadderOutcomes/"locker has no room for the item reward".
func TestGrantFailureRollsEverythingBack(t *testing.T) {
	db, tm, ctx, _ := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 500, 0)
	seedEmptyCompartment(t, db, ctx, testAccountId)
	// The currency reward is granted FIRST and the cash-item reward then fails,
	// so a partial commit would be visible as a credited wallet.
	id := seedCoupon(t, db, tm, NewBuilder("HALF").SetMaxUses(ptrU32(1)).SetRewards(reward.Rewards{
		reward.NewCurrencyReward(2, 1000),
		reward.NewCashItemReward(50200000, 1),
	}))
	cm := characterModel(t, testCharacterId, testAccountId)
	walletUpdates := countUpdatesTo(t, db, "accounts")

	if err := newProcessorForCharacter(t, ctx, db, cm, petCommodityItemId).RedeemAndEmit(testCharacterId, "HALF"); err == nil {
		t.Fatal("want a failure")
	}

	// Positive control: the wallet WAS written inside the transaction that then
	// failed. Without this the assertions below would hold vacuously for a run
	// that rejected the coupon before ever reaching the reward loop.
	if walletUpdates() == 0 {
		t.Fatal("no UPDATE was issued against the wallet; the failure happened before the reward loop, so this test proves nothing about rollback")
	}
	if p := loadWalletPoints(t, db, ctx, testAccountId); p != 500 {
		t.Errorf("points = %d, want the original 500 — the wallet must be untouched", p)
	}
	if n := countAssets(t, db); n != 0 {
		t.Errorf("locker assets = %d, want 0", n)
	}
	if n := loadCount(t, db, id); n != 0 {
		t.Errorf("redemptionCount = %d, want 0 — the reservation must be rolled back", n)
	}
	if n := countRedemptions(t, db, tm); n != 0 {
		t.Errorf("redemption rows = %d, want 0", n)
	}

	// And the code must still be redeemable: the rollback released the single
	// use it was holding. The commodity now resolves to a grantable item, which
	// is the only thing that changed.
	if err := newProcessorForCharacter(t, ctx, db, cm, couponRewardTemplateId).RedeemAndEmit(testCharacterId, "HALF"); err != nil {
		t.Fatalf("retry after rollback: %v — the code must still be redeemable", err)
	}
	if p := loadWalletPoints(t, db, ctx, testAccountId); p != 1500 {
		t.Errorf("points after the retry = %d, want 1500 (500 + the 1000 award)", p)
	}
	if n := countAssets(t, db); n != 1 {
		t.Errorf("locker assets after the retry = %d, want 1", n)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows after the retry = %d, want 1", n)
	}
}

// TestCaseAndWhitespaceVariantsAreTheSameCoupon: codes differing only in case
// or surrounding whitespace resolve to the same coupon, and redeeming one
// blocks the other for that account. No concurrency is involved, so this test
// is fully valid under this harness.
func TestCaseAndWhitespaceVariantsAreTheSameCoupon(t *testing.T) {
	db, tm, ctx, events := newProcessorTestEnv(t)
	seedWallet(t, db, ctx, testAccountId, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("MAPLE2026").SetRewards(reward.Rewards{reward.NewCurrencyReward(2, 10)}))
	cm := characterModel(t, testCharacterId, testAccountId)

	if err := newProcessorForCharacter(t, ctx, db, cm, couponRewardTemplateId).RedeemAndEmit(testCharacterId, " maple2026 "); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := newProcessorForCharacter(t, ctx, db, cm, couponRewardTemplateId).RedeemAndEmit(testCharacterId, "MaPlE2026"); err == nil {
		t.Fatal("second variant should have been rejected as already used")
	}
	if got := events.lastCouponFailure(); got != ErrorKeyAlreadyUsed {
		t.Errorf("key = %q, want %q", got, ErrorKeyAlreadyUsed)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows = %d, want 1", n)
	}
}
