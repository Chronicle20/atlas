package rest

import (
	"context"
	"time"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			if ts, err := reg.Get(ctx, suffix+":updatedAt"); err == nil && ts != "" {
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
			_ = reg.Remove(ctx, suffix)
			_ = reg.Remove(ctx, suffix+":updatedAt")
		}
	}
}

// jobRegistry is a convenience accessor that returns the JobCreator's Registry,
// or nil if either the JobCreator or its Registry is absent.
func (w Watchdog) jobRegistry() *redis.Registry[string, string] {
	if w.JobCreator == nil {
		return nil
	}
	return w.JobCreator.Registry
}
