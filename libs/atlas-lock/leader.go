package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bsm/redislock"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// Defaults exposed for documentation; consumers should pass options explicitly.
const (
	DefaultTTL             = 30 * time.Second
	DefaultRefreshInterval = 10 * time.Second // TTL / 3
	DefaultBackoff         = 5 * time.Second
	DefaultGracePeriod     = 5 * time.Second
)

type config struct {
	ttl             time.Duration
	refreshInterval time.Duration
	backoff         time.Duration
	gracePeriod     time.Duration
	log             logrus.FieldLogger
}

// Option mutates a config. Use the WithXxx constructors to obtain Options.
type Option func(*config)

// WithTTL sets the lease TTL. Allowed range: [5s, 5m]. Default: 30s.
func WithTTL(d time.Duration) Option { return func(c *config) { c.ttl = d } }

// WithRefreshInterval sets the renewal cadence. Allowed range: [1s, TTL/2]. Default: TTL/3.
func WithRefreshInterval(d time.Duration) Option {
	return func(c *config) { c.refreshInterval = d }
}

// WithBackoff sets the wait between failed acquire attempts. Allowed range: [1s, 1m]. Default: 5s.
func WithBackoff(d time.Duration) Option { return func(c *config) { c.backoff = d } }

// WithGracePeriod sets how long Run waits for fn to return after lease loss
// before logging a warning and proceeding. Allowed range: [1s, 30s]. Default: 5s.
func WithGracePeriod(d time.Duration) Option { return func(c *config) { c.gracePeriod = d } }

// WithLogger overrides the default logrus.New() logger.
func WithLogger(l logrus.FieldLogger) Option { return func(c *config) { c.log = l } }

func applyDefaults(c *config) {
	c.ttl = DefaultTTL
	c.refreshInterval = DefaultRefreshInterval
	c.backoff = DefaultBackoff
	c.gracePeriod = DefaultGracePeriod
	c.log = logrus.New()
}

const keyPrefix = "atlas:lock:"

// EnvVar names the deployment-environment discriminator baked into every lease
// key. Every Atlas service already carries it -- it is the same suffix that
// scopes Kafka topics (COMMAND_TOPIC_MIST-a628) and database names
// (atlas-maps-a628).
//
// Without it a lease name like "monsters-sweep" is GLOBAL across every
// deployment sharing a Redis, and every Atlas deployment does share one
// (REDIS_URL is redis.home:6379 in both atlas-main and each ephemeral
// atlas-pr-NNNN namespace, with no DB separation). The permanent main
// deployment wins the race at startup and holds the lease indefinitely, so a
// PR namespace's pod never becomes leader and silently runs NONE of its
// leader-gated work -- for atlas-monsters that is the whole sweep set:
// StatusExpirationTask (DoT ticks), drop timers, aggro decay, the skill
// picker sweep, monster recovery, and hidden reconciliation.
//
// Found in task-200 live testing: Poison Mist applied POISON to monsters
// correctly and their HP never moved, because atlas-pr-1255's atlas-monsters
// pod had never registered StatusExpirationTask. atlas-main's pod held
// atlas:lock:monsters-sweep.
const EnvVar = "ATLAS_ENV"

// unscopedEnv is the placeholder used when EnvVar is unset. It is a marker,
// not isolation: two deployments that both leave ATLAS_ENV unset and share a
// Redis still collide, and no key can separate them. Naming the shared bucket
// makes that visible in `redis-cli keys atlas:lock:*` instead of hiding it
// behind an omitted segment.
const unscopedEnv = "unscoped"

// scopeFromEnv reads the environment discriminator. Read once in New and
// stored on the LeaderElection so the key can never change under a running
// election -- an env mutation mid-flight would otherwise move the lease out
// from under the renewer while fn still believes it is leader.
func scopeFromEnv() string {
	v := strings.TrimSpace(os.Getenv(EnvVar))
	if v == "" {
		return unscopedEnv
	}
	return v
}

// LeaderElection runs a callback on exactly one pod for a named lease.
//
// Construction is cheap; only Run blocks. A single LeaderElection instance
// MUST NOT have Run called more than once concurrently. Construct one per
// logical role per pod.
type LeaderElection struct {
	rc   *goredis.Client
	name string
	// env is the deployment scope segment of the key, captured at New. See
	// scopeFromEnv.
	env string
	cfg config
}

// New constructs a LeaderElection bound to a Redis client and a service-scoped
// lease name. Returns an error for nil clients, empty/whitespace-only names,
// or option values outside the allowed ranges.
func New(rc *goredis.Client, name string, opts ...Option) (*LeaderElection, error) {
	if rc == nil {
		return nil, errors.New("lock: nil redis client")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("lock: name must be non-empty and not all-whitespace")
	}
	cfg := config{}
	applyDefaults(&cfg)
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.ttl < 5*time.Second || cfg.ttl > 5*time.Minute {
		return nil, fmt.Errorf("lock: TTL %s out of range [5s, 5m]", cfg.ttl)
	}
	if cfg.refreshInterval < time.Second || cfg.refreshInterval > cfg.ttl/2 {
		return nil, fmt.Errorf("lock: RefreshInterval %s out of range [1s, TTL/2]", cfg.refreshInterval)
	}
	if cfg.backoff < time.Second || cfg.backoff > time.Minute {
		return nil, fmt.Errorf("lock: Backoff %s out of range [1s, 1m]", cfg.backoff)
	}
	if cfg.gracePeriod < time.Second || cfg.gracePeriod > 30*time.Second {
		return nil, fmt.Errorf("lock: GracePeriod %s out of range [1s, 30s]", cfg.gracePeriod)
	}
	return &LeaderElection{rc: rc, name: name, env: scopeFromEnv(), cfg: cfg}, nil
}

// keyPath is the Redis lease key: prefix, deployment scope, then the
// service-scoped lease name. The scope segment is what keeps an ephemeral PR
// namespace from contending with the permanent deployment on a shared Redis
// -- see EnvVar.
func (le *LeaderElection) keyPath() string {
	return keyPrefix + le.env + ":" + le.name
}

// Run blocks until ctx is cancelled.
//
// While the lease is held by this pod, fn is invoked once with a child
// context. fn is expected to block on its leaderCtx until the lease is lost
// or the outer ctx is cancelled. On outer-ctx cancel, Run releases the lease
// (best-effort) and returns nil.
//
// A background renewer goroutine refreshes the lease every refreshInterval.
// If the lease cannot be renewed (ErrNotObtained), the inner leaderCtx is
// cancelled so fn can detect lease loss.
func (le *LeaderElection) Run(ctx context.Context, fn func(context.Context)) error {
	locker := redislock.New(le.rc)

	for {
		if ctx.Err() != nil {
			return nil
		}

		rl, err := locker.Obtain(ctx, le.keyPath(), le.cfg.ttl, &redislock.Options{
			RetryStrategy: redislock.NoRetry(),
		})
		if err != nil {
			if errors.Is(err, redislock.ErrNotObtained) {
				acquireFailedTotal.WithLabelValues(le.name, "held_by_other").Inc()
			} else {
				acquireFailedTotal.WithLabelValues(le.name, "redis_error").Inc()
				le.cfg.log.WithError(err).Debugf("Acquire for [%s] failed: %v", le.name, err)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(le.cfg.backoff):
			}
			continue
		}

		acquiredTotal.WithLabelValues(le.name).Inc()
		le.cfg.log.Infof("Acquired leader for [%s].", le.name)

		leaderCtx, cancelLeader := context.WithCancel(ctx)
		fnDone := make(chan struct{})
		renewerDone := make(chan struct{})

		// First-writer-wins reason; multiple goroutines call setReason.
		var lostReason atomic.Value // string
		setReason := func(r string) { lostReason.CompareAndSwap(nil, r) }

		routine.Go(le.cfg.log, leaderCtx, func(c context.Context) {
			defer close(fnDone)
			completed := false
			defer func() {
				if !completed {
					setReason("panic")
					cancelLeader()
				}
			}()
			fn(c)
			completed = true
		})

		routine.Go(le.cfg.log, leaderCtx, func(_ context.Context) {
			defer close(renewerDone)
			t := time.NewTicker(le.cfg.refreshInterval)
			defer t.Stop()
			for {
				select {
				case <-leaderCtx.Done():
					return
				case <-t.C:
					rerr := rl.Refresh(ctx, le.cfg.ttl, nil)
					if rerr == nil {
						continue
					}
					if errors.Is(rerr, redislock.ErrNotObtained) {
						setReason("renew_failed")
						le.cfg.log.WithError(rerr).Warnf("Lease lost during refresh for [%s].", le.name)
						cancelLeader()
						return
					}
					renewFailedTotal.WithLabelValues(le.name).Inc()
					le.cfg.log.WithError(rerr).Warnf("Renewal attempt failed for [%s] (transient).", le.name)
				}
			}
		})

		select {
		case <-ctx.Done():
			setReason("context_cancelled")
		case <-fnDone:
			setReason("released")
		}
		cancelLeader()
		<-renewerDone

		graceTimer := time.NewTimer(le.cfg.gracePeriod)
		select {
		case <-fnDone:
			if !graceTimer.Stop() {
				<-graceTimer.C
			}
		case <-graceTimer.C:
			le.cfg.log.Warnf("Leader fn did not return within grace period [%s] for [%s]; proceeding without waiting.",
				le.cfg.gracePeriod, le.name)
		}

		relCtx, relCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = rl.Release(relCtx)
		relCancel()

		reason, _ := lostReason.Load().(string)
		if reason == "" {
			reason = "released"
		}
		lostTotal.WithLabelValues(le.name, reason).Inc()
		le.cfg.log.Infof("Lost leader for [%s] (reason: %s).", le.name, reason)

		if ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(le.cfg.backoff):
		}
	}
}
