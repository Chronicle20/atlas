package parcel

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	// DefaultExpiryInterval is the sweep cadence when no interval is
	// supplied (main.go falls back to this when
	// PARCEL_EXPIRY_INTERVAL_SECONDS is unset/invalid).
	DefaultExpiryInterval = 1 * time.Hour

	// defaultExpiryBatch bounds how many expired parcels a single Run()
	// claims — a batch cap on the same shape as atlas-mts's
	// sweepBatchLimit. The remainder is picked up on the next tick, never
	// silently truncated.
	defaultExpiryBatch = 500

	// returnMessage is the server-authored message stamped on a return leg
	// (design §7.4, OQ-7) — there is no wire field for "this is a return",
	// so the distinction is carried by this message plus the swapped
	// sender identity.
	returnMessage = "Unclaimed parcel returned."
)

// ExpiryTask is the background sweep that expires unclaimed pending
// parcels and, for the ones that were never themselves a return leg,
// inserts a return-to-sender leg (design §7.4 / §8). It mirrors
// services/atlas-merchant/atlas.com/merchant/frederick/task.go's Run()/
// SleepTime() pair for its DB-driven claim work, and
// services/atlas-mts/atlas.com/mts/task/periodic.go's ticker/Start/Stop
// shape for how it is driven in production.
type ExpiryTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
	batch    int
	now      func() time.Time
	stopCh   chan struct{}
	wg       *sync.WaitGroup
}

// NewExpiryTask constructs the sweep. A non-positive interval falls back to
// DefaultExpiryInterval.
func NewExpiryTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *ExpiryTask {
	if interval <= 0 {
		interval = DefaultExpiryInterval
	}
	l.Infof("Initializing parcel expiry sweep to run every %dms.", interval.Milliseconds())
	return &ExpiryTask{
		l:        l,
		ctx:      ctx,
		db:       db,
		interval: interval,
		batch:    defaultExpiryBatch,
		now:      time.Now,
		stopCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

// withClock returns a copy of the task with now replaced — unexported,
// mirroring ProcessorImpl.withClock. Only this package's own tests use it.
func (t *ExpiryTask) withClock(now func() time.Time) *ExpiryTask {
	c := *t
	c.now = now
	return &c
}

// withBatch returns a copy of the task with its claim batch size replaced —
// unexported, test-only seam for exercising the batch cap (Step 2's "batch
// bound" subtest) without waiting for defaultExpiryBatch rows.
func (t *ExpiryTask) withBatch(batch int) *ExpiryTask {
	c := *t
	c.batch = batch
	return &c
}

// SleepTime reports the sweep's cadence.
func (t *ExpiryTask) SleepTime() time.Duration {
	return t.interval
}

// Start launches the ticker loop.
func (t *ExpiryTask) Start() {
	t.wg.Add(1)
	routine.Go(t.l, t.ctx, func(context.Context) { t.run() })
	t.l.Infof("Parcel expiry sweep started with interval [%v].", t.interval)
}

// Stop signals the loop to exit and waits for the in-flight tick to finish.
func (t *ExpiryTask) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	t.l.Infoln("Parcel expiry sweep stopped.")
}

func (t *ExpiryTask) run() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.Run()
		case <-t.stopCh:
			return
		}
	}
}

// Run performs one claim-and-return pass: it claims up to t.batch
// pending, past-ExpiresAt parcels across every tenant (ClaimExpired, design
// §8.1) and, for each claimed row that was NOT itself a return leg, inserts
// a return-to-sender leg back in that row's own tenant context.
//
// A claimed row with Returned() == true is a return leg that itself
// expired unclaimed — it is destroyed and creates nothing (design §7.4),
// which is exactly what the Returned flag exists to guarantee: without it,
// a return leg's own expiry would recurse forever.
//
// RecipientName sourcing (a named, in-progress gap): the return leg's
// SenderName is the ORIGINAL parcel's RecipientName — the person who never
// claimed it. That field is not yet populated by the currently-landed
// atlas-channel send saga (Task 17's TransferToParcelPayload/
// AcceptToParcelPayload carry no such field), so today every claimed row's
// RecipientName is "" and every return leg's SenderName will be empty
// rather than a display name, until a follow-up task threads it through
// libs/atlas-saga's TransferToParcelPayload and AcceptToParcelPayload, the
// orchestrator's TransferToParcel expansion, and this service's own
// custody.AcceptToParcelCommandBody. See entity.go's RecipientName doc
// comment for the exact wiring. The sweep itself is correct and complete
// for every other field.
func (t *ExpiryTask) Run() {
	noTenantCtx := database.WithoutTenantFilter(t.ctx)
	now := t.now()
	tdb := t.db.WithContext(noTenantCtx)

	claimed, err := ClaimExpired(tdb)(now, t.batch)
	if err != nil {
		t.l.WithError(err).Errorln("Parcel expiry sweep: failed to claim expired parcels.")
		return
	}
	if len(claimed) == 0 {
		return
	}

	returned := 0
	for _, m := range claimed {
		if m.Returned() {
			// A return leg's own expiry produces nothing (design §7.4).
			continue
		}

		tm, terr := tenant.Create(m.TenantId(), "", 0, 0)
		if terr != nil {
			t.l.WithError(terr).Warnf("Parcel expiry sweep: failed to reconstruct tenant [%s] for expired parcel [%s]; no return leg created.", m.TenantId(), m.Id())
			continue
		}
		// Deliberately built from t.ctx, NOT noTenantCtx: WithoutTenantFilter
		// also disables the tenant:create callback's tenant_id injection
		// (database/tenant_scope.go), which would otherwise leave the
		// return leg's tenant_id at its zero value and make it invisible
		// to every normal, tenant-scoped read.
		tenantCtx := tenant.WithContext(t.ctx, tm)

		rm, berr := NewBuilder().
			SetId(uuid.New()).
			SetTenantId(m.TenantId()).
			SetWorldId(m.WorldId()).
			SetSenderId(m.RecipientId()).
			SetSenderAccountId(m.RecipientAccountId()).
			SetSenderName(m.RecipientName()).
			SetRecipientId(m.SenderId()).
			SetRecipientAccountId(m.SenderAccountId()).
			SetRecipientName(m.SenderName()).
			SetMessage(returnMessage).
			SetMesoAmount(m.MesoAmount()).
			SetFeePaid(0).
			SetItemId(m.ItemId()).
			SetItemType(m.ItemType()).
			SetQuantity(m.Quantity()).
			SetItemSnapshot(m.ItemSnapshot()).
			SetStatus(StatusPending).
			SetQuick(false).
			SetReturned(true).
			SetCreatedAt(now).
			SetReceivableAt(now).
			SetExpiresAt(now.Add(ExpiryWindow)).
			Build()
		if berr != nil {
			t.l.WithError(berr).Warnf("Parcel expiry sweep: failed to build return leg for expired parcel [%s]; no return leg created.", m.Id())
			continue
		}

		if _, cerr := Create(t.db.WithContext(tenantCtx))(rm); cerr != nil {
			t.l.WithError(cerr).Warnf("Parcel expiry sweep: failed to create return leg for expired parcel [%s] (tenant [%s]).", m.Id(), m.TenantId())
			continue
		}
		returned++
	}

	t.l.Infof("Parcel expiry sweep: expired [%d] parcel(s), created [%d] return leg(s).", len(claimed), returned)
}
