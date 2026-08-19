package parcel

import (
	parcelmsg "atlas-parcel/kafka/message/parcel"
	parcelproducer "atlas-parcel/kafka/producer/parcel"
	"context"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	// DefaultNotificationInterval is the sweep cadence when no interval is
	// supplied (main.go falls back to this when
	// PARCEL_NOTIFICATION_INTERVAL_SECONDS is unset/invalid). A 30-day
	// expiry deadline does not need a tight sweep, but an arrival
	// notification does (design §8.2) — five minutes, not ExpiryTask's
	// hour.
	DefaultNotificationInterval = 5 * time.Minute

	// defaultNotificationBatch bounds how many newly-receivable parcels a
	// single Run() claims — the same batch-cap shape as
	// defaultExpiryBatch (task.go): the remainder is picked up on the next
	// tick, never silently truncated.
	defaultNotificationBatch = 500
)

// NotificationTask is the background sweep that notifies recipients of
// newly-receivable parcels via a PARCEL_ARRIVED status event and stamps
// LastNotified so it is sent at most once per parcel (design §7.1, §8). It
// mirrors ExpiryTask's Run()/SleepTime()/Start()/Stop() shape (task.go) and
// services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go's
// envContext re-entry before any Kafka emit.
type NotificationTask struct {
	l          logrus.FieldLogger
	ctx        context.Context
	db         *gorm.DB
	interval   time.Duration
	envContext func(context.Context) context.Context
	batch      int
	now        func() time.Time
	stopCh     chan struct{}
	wg         *sync.WaitGroup
}

// NewNotificationTask constructs the sweep. A non-positive interval falls
// back to DefaultNotificationInterval. envContext originates this pod's own
// environment identity onto each claimed parcel's per-tenant context before
// the Kafka emit — an empty ENVIRONMENT header would make decide() fail open
// per FR-1.8 and every live deployment, not just this pod's, would notify
// (frederick's processDueNotifications carries the same rationale).
func NewNotificationTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *NotificationTask {
	if interval <= 0 {
		interval = DefaultNotificationInterval
	}
	l.Infof("Initializing parcel notification sweep to run every %dms.", interval.Milliseconds())
	return &NotificationTask{
		l:          l,
		ctx:        ctx,
		db:         db,
		interval:   interval,
		envContext: envContext,
		batch:      defaultNotificationBatch,
		now:        time.Now,
		stopCh:     make(chan struct{}),
		wg:         &sync.WaitGroup{},
	}
}

// withClock returns a copy of the task with now replaced — unexported,
// mirroring ExpiryTask.withClock. Only this package's own tests use it.
func (t *NotificationTask) withClock(now func() time.Time) *NotificationTask {
	c := *t
	c.now = now
	return &c
}

// withBatch returns a copy of the task with its claim batch size replaced —
// unexported, test-only seam mirroring ExpiryTask.withBatch.
func (t *NotificationTask) withBatch(batch int) *NotificationTask {
	c := *t
	c.batch = batch
	return &c
}

// SleepTime reports the sweep's cadence.
func (t *NotificationTask) SleepTime() time.Duration {
	return t.interval
}

// Start launches the ticker loop.
func (t *NotificationTask) Start() {
	t.wg.Add(1)
	routine.Go(t.l, t.ctx, func(context.Context) { t.run() })
	t.l.Infof("Parcel notification sweep started with interval [%v].", t.interval)
}

// Stop signals the loop to exit and waits for the in-flight tick to finish.
func (t *NotificationTask) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	t.l.Infoln("Parcel notification sweep stopped.")
}

func (t *NotificationTask) run() {
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

// Run performs one claim-and-notify pass: it checks the status event topic
// is configured, then claims up to t.batch newly-receivable, not-yet-notified
// parcels across every tenant (ClaimNotifiable, design §8.1) — the claim's
// UPDATE stamps LastNotified as part of the SAME statement that selects the
// row, so the topic check happens BEFORE the claim: an unconfigured topic
// must not stamp a parcel it never actually notified. For each claimed row,
// it reconstructs that row's tenant, re-enters it (with envContext layered
// on top, load-bearing ordering — see NewNotificationTask), and emits
// PARCEL_ARRIVED. Offline recipients are still notified and still stamped —
// FR-24 is served by the OPEN packet's second list, not by this sweep
// (design §7.1).
//
// The topic-configured check is a direct os.LookupEnv, not
// topic.EnvProvider: that provider always falls back to the token itself and
// never actually errors on a missing var (libs/atlas-kafka/topic/provider.go),
// so it cannot express "operator has not wired this topic yet, don't run."
func (t *NotificationTask) Run() {
	if _, ok := os.LookupEnv(parcelmsg.EnvStatusEventTopic); !ok {
		t.l.Warnf("Parcel status event topic [%s] not configured, skipping notifications.", parcelmsg.EnvStatusEventTopic)
		return
	}

	noTenantCtx := database.WithoutTenantFilter(t.ctx)
	now := t.now()
	tdb := t.db.WithContext(noTenantCtx)

	claimed, err := ClaimNotifiable(tdb)(now, t.batch)
	if err != nil {
		t.l.WithError(err).Errorln("Parcel notification sweep: failed to claim notifiable parcels.")
		return
	}
	if len(claimed) == 0 {
		return
	}

	notified := 0
	for _, m := range claimed {
		tm, terr := tenant.Create(m.TenantId(), "", 0, 0)
		if terr != nil {
			t.l.WithError(terr).Warnf("Parcel notification sweep: failed to reconstruct tenant [%s] for parcel [%s]; notification skipped.", m.TenantId(), m.Id())
			continue
		}
		tenantCtx := t.envContext(tenant.WithContext(t.ctx, tm))

		kp := producer.ProviderImpl(t.l)(tenantCtx)
		if perr := kp(parcelmsg.EnvStatusEventTopic)(parcelproducer.ParcelArrivedStatusEventProvider(m.RecipientId(), m.SenderName(), m.ItemId() != nil)); perr != nil {
			t.l.WithError(perr).Warnf("Parcel notification sweep: failed to emit PARCEL_ARRIVED for parcel [%s].", m.Id())
			continue
		}
		notified++
	}

	t.l.Infof("Parcel notification sweep: notified [%d] of [%d] claimed parcel(s).", notified, len(claimed))
}
