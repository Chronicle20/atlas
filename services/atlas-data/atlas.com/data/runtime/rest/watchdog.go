package rest

import (
	"atlas-data/ingestrun"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// DefaultWatchdogTimeoutSecs is the maximum heartbeat staleness the Watchdog
// tolerates before deleting an ingest Job, and — because the status handler
// uses the same window to decide whether a `running` record is corroborated —
// the single definition of "how stale is too stale". The ingest pod refreshes
// its heartbeat every 30s (runtime/ingest/heartbeat.go), so anything over ~60s
// suffices in the happy path; 2 h is a generous margin for a wedged heartbeat
// goroutine or a transient Redis blip, and absorbs archive growth without a
// code change.
const DefaultWatchdogTimeoutSecs = 7200

// Watchdog periodically sweeps the set of active ingest Jobs and deletes any
// that have exceeded TimeoutSecs without progress, removing the corresponding
// Redis heartbeat keys.
type Watchdog struct {
	L           logrus.FieldLogger
	JobCreator  *JobCreator
	TimeoutSecs int
}

// Run blocks until ctx is cancelled, ticking once every 30 seconds.
func (w Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweep(ctx)
		}
	}
}

// sweep lists all ingest Jobs and deletes those whose heartbeat (or, in the
// absence of a heartbeat, creation timestamp) is older than TimeoutSecs.
func (w Watchdog) sweep(ctx context.Context) {
	if w.JobCreator == nil || w.JobCreator.K8s == nil {
		return
	}
	if w.TimeoutSecs <= 0 {
		return
	}
	list, err := w.JobCreator.K8s.BatchV1().Jobs(w.JobCreator.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelIngest + "=true",
	})
	if err != nil {
		if w.L != nil {
			w.L.WithError(err).Warn("watchdog: list jobs failed")
		}
		return
	}
	cutoff := time.Now().Add(-time.Duration(w.TimeoutSecs) * time.Second)
	for i := range list.Items {
		j := &list.Items[i]
		// Already finished — nothing to watchdog.
		if j.Status.Succeeded > 0 || jobFailed(j) {
			continue
		}
		if w.jobIsStuck(ctx, j, cutoff) {
			w.deleteStuckJob(ctx, j)
		}
	}
}

// jobFailed reports whether the Job's status carries a Failed condition.
func jobFailed(j *batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == "True" {
			return true
		}
	}
	return false
}

// jobIsStuck returns true if the most recent heartbeat (or, lacking a
// heartbeat, the Job's creation timestamp) is older than cutoff.
func (w Watchdog) jobIsStuck(ctx context.Context, j *batchv1.Job, cutoff time.Time) bool {
	ref := j.CreationTimestamp.Time
	if reg := w.jobRegistry(); reg != nil {
		if suffix := ingestJobKeySuffixFromLabels(j); suffix != "" {
			if ts, err := reg.Get(ctx, env.Self(), suffix+ingestrun.HeartbeatKeySuffix); err == nil && ts != "" {
				if t, perr := time.Parse(time.RFC3339, ts); perr == nil {
					ref = t
				}
			}
		}
	}
	return ref.Before(cutoff)
}

// deleteStuckJob deletes the Job in k8s and drops the heartbeat keys in Redis.
func (w Watchdog) deleteStuckJob(ctx context.Context, j *batchv1.Job) {
	if w.L != nil {
		w.L.Warnf("watchdog: job %s stuck, deleting", j.Name)
	}
	policy := metav1.DeletePropagationForeground
	if err := w.JobCreator.K8s.BatchV1().Jobs(w.JobCreator.Namespace).Delete(ctx, j.Name, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}); err != nil && w.L != nil {
		w.L.WithError(err).Warnf("watchdog: delete job %s failed", j.Name)
	}
	if reg := w.jobRegistry(); reg != nil {
		if suffix := ingestJobKeySuffixFromLabels(j); suffix != "" {
			_ = reg.Remove(ctx, env.Self(), suffix)
			_ = reg.Remove(ctx, env.Self(), suffix+ingestrun.HeartbeatKeySuffix)
		}
	}
	if rr := w.runRegistry(); rr != nil {
		if suffix := ingestJobKeySuffixFromLabels(j); suffix != "" {
			reason := fmt.Sprintf("watchdog deleted the ingest Job after %ds without a heartbeat", w.TimeoutSecs)
			now := time.Now().UTC()
			runId := ingestRunIdFromJob(j)
			// Per-worker states are left exactly as the ingest pod wrote them:
			// the worker still `running` when the watchdog fired is the whole
			// diagnostic value, and marking it failed would assert something we
			// do not know.
			_, err := rr.UpdateWithTTL(ctx, env.Self(), suffix+ingestrun.RunKeySuffix, ingestrun.RecordTTL,
				func(rec ingestrun.Record) ingestrun.Record {
					// Guarded exactly like every ingest-pod write (see
					// runtime/ingest/progress.go's guardedUpdate): a sweep that
					// lags a re-triggered run for the same (scope, region,
					// version) must not stamp phase=stuck over a newer run's
					// live record. Re-evaluated on every optimistic-lock retry
					// so a stale sweep can never win the race. An empty runId
					// on either side (older record predating run ids, or a Job
					// whose env we couldn't read) means the guard cannot
					// decide, and the write is allowed through.
					if rec.RunId != "" && runId != "" && rec.RunId != runId {
						return rec
					}
					return rec.WithPhase(ingestrun.PhaseStuck, now, reason)
				})
			if err != nil && !errors.Is(err, redis.ErrNotFound) && w.L != nil {
				w.L.WithError(err).Warnf("watchdog: stuck-record write failed for %s", suffix)
			}
		}
	}
}

// ingestRunIdFromJob recovers the run id the JobCreator stamped onto the Job
// at creation time. Job objects carry no run-id label (jobs.go's renderJob
// only labels scope/region/version/tenant); the run id instead rides as the
// INGEST_RUN_ID env var injected into every container (jobs.go:292-294), so
// that is what the Watchdog reads back to guard its stuck-record write.
// Returns "" if no container carries the var.
func ingestRunIdFromJob(j *batchv1.Job) string {
	for _, c := range j.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == "INGEST_RUN_ID" {
				return e.Value
			}
		}
	}
	return ""
}

// jobRegistry is a convenience accessor that returns the JobCreator's Registry,
// or nil if either the JobCreator or its Registry is absent.
func (w Watchdog) jobRegistry() *redis.EnvironmentRegistry[string, string] {
	if w.JobCreator == nil {
		return nil
	}
	return w.JobCreator.Registry
}

// runRegistry returns the JobCreator's run-record Registry, or nil if either
// the JobCreator or its registry is absent.
func (w Watchdog) runRegistry() *redis.EnvironmentRegistry[string, ingestrun.Record] {
	if w.JobCreator == nil {
		return nil
	}
	return w.JobCreator.RunRegistry
}
