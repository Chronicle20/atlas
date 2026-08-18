package scheduling

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// Default Config values applied to any zero-valued field, mirroring
// libs/atlas-outbox's NewDrainer option-default pattern.
const (
	defaultInterval  = 5 * time.Second
	defaultLease     = 5 * time.Minute
	defaultBatchSize = 50
	// defaultPollerBackoff is Config.Backoff's production default. This is
	// distinct from processor.go's defaultBackoff (0, immediate retry), which
	// applies only to a bare NewProcessor built without a Config — every
	// Poller-driven processor gets this non-zero delay instead.
	defaultPollerBackoff = 30 * time.Second
)

// Config carries the poller's tunables, read from env by main.go (FR-N16,
// Task 20).
type Config struct {
	// Interval is how often the poller ticks: claim a batch, execute it.
	Interval time.Duration
	// Lease is how long a claimed row may sit PROCESSING before Reclaim
	// treats its claimer as dead and returns it to PENDING (FR-S7).
	Lease time.Duration
	// BatchSize is the maximum number of rows ClaimBatch takes per tick.
	BatchSize int
	// MaxAttempts is the retry ceiling ExecuteOne's outcome policy applies
	// (design §5.2, FR-S9).
	MaxAttempts int
	// Backoff is the delay ExecuteOne applies to a row that errored and
	// still has attempts remaining.
	Backoff time.Duration
}

// withDefaults fills any zero-valued field with its production default.
func (c Config) withDefaults() Config {
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	if c.Lease <= 0 {
		c.Lease = defaultLease
	}
	if c.BatchSize <= 0 {
		c.BatchSize = defaultBatchSize
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaultMaxAttempts
	}
	if c.Backoff <= 0 {
		c.Backoff = defaultPollerBackoff
	}
	return c
}

// Poller is a plain ticker over ClaimBatch + Reclaim (design §5.1, §5.2). It
// is NOT leader-elected — every replica polls, and SKIP LOCKED isolates a
// stuck row to its own claimer instead of stalling the whole scheduler
// (FR-N6). instanceId distinguishes this replica's claims from any other's;
// callers should pass something stable per-process (hostname, pod name).
type Poller struct {
	l          logrus.FieldLogger
	instanceId string
	p          *ProcessorImpl
	cfg        Config
	stop       chan struct{}
}

// NewPoller constructs a Poller. Its Processor is configured from cfg's
// MaxAttempts/Backoff so ExecuteOne's outcome policy matches the poller's own
// tunables. instanceId (the claimer identity ClaimBatch stamps onto claimed
// rows) is derived from the process's hostname plus a random suffix so two
// replicas on the same host still claim under distinct identities; a
// hostname lookup failure falls back to a bare UUID.
func NewPoller(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, cfg Config) *Poller {
	cfg = cfg.withDefaults()
	p := NewProcessor(l, ctx, db)
	p.SetMaxAttempts(cfg.MaxAttempts)
	p.SetBackoff(cfg.Backoff)
	return &Poller{
		l:          l,
		instanceId: instanceId(),
		p:          p,
		cfg:        cfg,
		stop:       make(chan struct{}),
	}
}

// ConfigFromEnv reads the poller's tunables from the environment (FR-N16,
// Task 20). Each var falls back to its production default (withDefaults)
// when unset or unparseable, matching the atlas-saga-orchestrator reaper's
// os.LookupEnv + best-effort-parse idiom:
//
//   - EVENTS_POLL_INTERVAL     time.Duration string, e.g. "5s" (default 5s)
//   - EVENTS_WORK_LEASE        time.Duration string, e.g. "5m" (default 5m)
//   - EVENTS_POLL_BATCH_SIZE   integer (default 50)
//   - EVENTS_WORK_MAX_ATTEMPTS integer (default 5)
//   - EVENTS_WORK_BACKOFF      time.Duration string, e.g. "30s" (default 30s)
func ConfigFromEnv() Config {
	var cfg Config
	if v, ok := os.LookupEnv("EVENTS_POLL_INTERVAL"); ok {
		if parsed, err := time.ParseDuration(v); err == nil {
			cfg.Interval = parsed
		}
	}
	if v, ok := os.LookupEnv("EVENTS_WORK_LEASE"); ok {
		if parsed, err := time.ParseDuration(v); err == nil {
			cfg.Lease = parsed
		}
	}
	if v, ok := os.LookupEnv("EVENTS_POLL_BATCH_SIZE"); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.BatchSize = parsed
		}
	}
	if v, ok := os.LookupEnv("EVENTS_WORK_MAX_ATTEMPTS"); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.MaxAttempts = parsed
		}
	}
	if v, ok := os.LookupEnv("EVENTS_WORK_BACKOFF"); ok {
		if parsed, err := time.ParseDuration(v); err == nil {
			cfg.Backoff = parsed
		}
	}
	return cfg.withDefaults()
}

// instanceId names this replica for ClaimBatch's claimed_by column.
func instanceId() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return uuid.New().String()
	}
	return host + "-" + uuid.New().String()
}

// Run ticks on cfg.Interval, claiming and executing a batch each tick. It
// calls Reclaim once immediately, before the first tick, so work orphaned by
// the previous process's death resumes promptly rather than after a full
// lease interval (design §6). It blocks until ctx is done or Stop is called,
// so callers invoke it via routine.Go.
func (poller *Poller) Run(ctx context.Context) {
	poller.reclaimOnce(ctx)
	t := time.NewTicker(poller.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-poller.stop:
			return
		case <-t.C:
			poller.tickOnce(ctx)
		}
	}
}

// SetOwnership installs the environment-ownership predicate on the poller's
// processor (task-232). Unset, the poller behaves exactly as it did before
// sparse ephemeral environments existed: it owns every tenant it can see.
func (poller *Poller) SetOwnership(o TenantOwnership) { poller.p.SetOwnership(o) }

// Stop halts Run's loop. Idempotent within a single close; callers that need
// to stop more than once should guard their own call site.
func (poller *Poller) Stop() {
	close(poller.stop)
}

// Start launches Run in a background goroutine via routine.Go, recovering
// any panic so one contained failure cannot take the whole process down.
func (poller *Poller) Start(ctx context.Context) {
	routine.Go(poller.l, ctx, poller.Run)
}

func (poller *Poller) reclaimOnce(ctx context.Context) {
	n, err := poller.p.Reclaim(poller.cfg.Lease)
	if err != nil {
		poller.l.WithError(err).Warn("scheduling.reclaim_failed")
		return
	}
	if n > 0 {
		poller.l.WithField("reclaimed", n).Info("scheduling.reclaimed_orphaned_work")
	}
}

func (poller *Poller) tickOnce(ctx context.Context) {
	poller.reclaimOnce(ctx)

	claimed, err := poller.p.ClaimBatch(poller.instanceId, poller.cfg.BatchSize)
	if err != nil {
		poller.l.WithError(err).Warn("scheduling.claim_failed")
		return
	}
	for _, m := range claimed {
		if err := poller.p.ExecuteOne(m); err != nil {
			poller.l.WithError(err).WithField("workId", m.Id().String()).
				Warn("scheduling.execute_failed")
		}
	}
}
