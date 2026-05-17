package outbox

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Publisher interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type PublisherFunc func(ctx context.Context, msgs ...kafka.Message) error

func (f PublisherFunc) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return f(ctx, msgs...)
}

type drainerConfig struct {
	pollInterval    time.Duration
	batchSize       int
	sweeperInterval time.Duration
	retention       time.Duration
	dsn             string
}

type DrainerOption func(*drainerConfig)

func WithPollInterval(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.pollInterval = d }
}

func WithBatchSize(n int) DrainerOption {
	return func(c *drainerConfig) { c.batchSize = n }
}

func WithSweeperInterval(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.sweeperInterval = d }
}

func WithRetention(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.retention = d }
}

// WithDSN supplies a PostgreSQL DSN used to open a dedicated LISTEN
// connection (via pq.Listener) so the leader wakes immediately when
// Enqueue calls pg_notify, instead of waiting for the next poll tick.
// When unset, the drainer falls back to ticker-only polling.
func WithDSN(dsn string) DrainerOption {
	return func(c *drainerConfig) { c.dsn = dsn }
}

type Drainer struct {
	l    logrus.FieldLogger
	db   *gorm.DB
	pub  Publisher
	cfg  drainerConfig
	stop chan struct{}
	ntfy *notifier
}

func NewDrainer(l logrus.FieldLogger, db *gorm.DB, pub Publisher, opts ...DrainerOption) *Drainer {
	cfg := drainerConfig{
		pollInterval:    1 * time.Second,
		batchSize:       100,
		sweeperInterval: 1 * time.Hour,
		retention:       7 * 24 * time.Hour,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Drainer{l: l, db: db, pub: pub, cfg: cfg, stop: make(chan struct{})}
}

func (d *Drainer) Run(ctx context.Context) {
	d.ensureNotifier()
	defer func() {
		if d.ntfy != nil {
			d.ntfy.Close()
			d.ntfy = nil
		}
	}()
	// Attempt an immediate lock acquisition before entering the poll loop
	// so a cold-start leader doesn't sit idle for a full poll interval
	// before draining (and so NOTIFY wakeups land on an active leader).
	d.tickOnce(ctx)
	t := time.NewTicker(d.cfg.pollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-t.C:
			d.tickOnce(ctx)
		}
	}
}

func (d *Drainer) tickOnce(ctx context.Context) {
	if !isPostgres(d.db) {
		_ = d.publishBatch(ctx)
		return
	}
	lk, err := tryAdvisoryLock(ctx, d.db)
	if err != nil {
		d.l.WithError(err).Warn("outbox.lock_acquire_error")
		return
	}
	if !lk.Held() {
		return
	}
	d.l.Info("outbox.lock_acquired")
	d.runLeader(ctx)
	lk.Release(context.Background())
	d.l.Info("outbox.lock_lost")
}

// ensureNotifier opens the LISTEN connection eagerly at Run start so that
// NOTIFY signals emitted before leadership is acquired still land in the
// buffered channel and wake the new leader on the first runLeader entry.
func (d *Drainer) ensureNotifier() {
	if d.cfg.dsn == "" || !isPostgres(d.db) || d.ntfy != nil {
		return
	}
	n, err := newNotifier(d.l, d.cfg.dsn)
	if err != nil {
		d.l.WithError(err).Warn("outbox.notify_listen_failed")
		return
	}
	d.ntfy = n
}

func (d *Drainer) runLeader(ctx context.Context) {
	// Drain any rows that accumulated before this replica became leader.
	if err := d.publishBatch(ctx); err != nil {
		d.l.WithError(err).Warn("outbox.publish_failed")
		return
	}
	// Sweeper runs only while leader; cancel on leader exit.
	sweepCtx, cancelSweep := context.WithCancel(ctx)
	defer cancelSweep()
	go d.runSweeper(sweepCtx)

	tk := time.NewTicker(d.cfg.pollInterval)
	defer tk.Stop()
	var notifyCh <-chan struct{}
	if d.ntfy != nil {
		notifyCh = d.ntfy.C()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-tk.C:
			if err := d.publishBatch(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.publish_failed")
				return
			}
		case <-notifyCh:
			if err := d.publishBatch(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.publish_failed")
				return
			}
		}
	}
}

func (d *Drainer) Stop() { close(d.stop) }

// SweepOnce deletes published rows whose sent_at is older than the
// configured retention window. The drainer schedules this on its own
// cadence; it is exposed for tests and for operator-driven sweeps.
func (d *Drainer) SweepOnce(ctx context.Context) error {
	cutoff := time.Now().Add(-d.cfg.retention)
	res := d.db.WithContext(ctx).Where("sent_at IS NOT NULL AND sent_at < ?", cutoff).Delete(&Entity{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		d.l.WithField("deleted", res.RowsAffected).Info("outbox.sweeper_run")
	}
	return nil
}

// runSweeper is launched from Run; it ticks on cfg.sweeperInterval and
// invokes SweepOnce. Leader-only is enforced by the caller (Run only
// starts this when leadership is held).
func (d *Drainer) runSweeper(ctx context.Context) {
	tk := time.NewTicker(d.cfg.sweeperInterval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-tk.C:
			if err := d.SweepOnce(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.sweeper_failed")
			}
		}
	}
}

func (d *Drainer) publishBatch(ctx context.Context) error {
	var failedIDs []uint64
	var failedErr error
	txErr := d.db.Transaction(func(tx *gorm.DB) error {
		var rows []Entity
		if isPostgres(d.db) {
			q := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
			if err := q.Where("sent_at IS NULL").Order("enqueued_at ASC").Limit(d.cfg.batchSize).Find(&rows).Error; err != nil {
				return err
			}
		} else {
			if err := tx.WithContext(ctx).Where("sent_at IS NULL").Order("enqueued_at ASC").Limit(d.cfg.batchSize).Find(&rows).Error; err != nil {
				return err
			}
		}
		if len(rows) == 0 {
			return nil
		}

		msgs := make([]kafka.Message, 0, len(rows))
		for _, r := range rows {
			msgs = append(msgs, kafka.Message{
				Topic: r.Topic,
				Key:   r.MessageKey,
				Value: r.MessageValue,
			})
		}
		if err := d.pub.WriteMessages(ctx, msgs...); err != nil {
			ids := make([]uint64, 0, len(rows))
			for _, r := range rows {
				ids = append(ids, r.ID)
			}
			// Defer failure bookkeeping until after the SELECT/UPDATE
			// transaction rolls back, otherwise an update against the still-
			// locked rows from a second pool connection would self-deadlock.
			failedIDs = ids
			failedErr = err
			return err
		}

		now := time.Now()
		ids := make([]uint64, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return tx.WithContext(ctx).
			Model(&Entity{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"sent_at": &now}).Error
	})
	if failedErr != nil && len(failedIDs) > 0 {
		d.db.WithContext(ctx).
			Model(&Entity{}).
			Where("id IN ?", failedIDs).
			Updates(map[string]any{"attempts": gorm.Expr("attempts + 1"), "last_error": failedErr.Error()})
	}
	return txErr
}
