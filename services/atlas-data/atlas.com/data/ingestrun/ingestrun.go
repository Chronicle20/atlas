// Package ingestrun holds the shared per-run progress record for atlas-data's
// WZ ingest, plus the Redis key/namespace helpers both runtime modes use.
//
// The record is written by runtime/rest (Job creation, Watchdog) and by
// runtime/ingest (per-worker transitions), and read by runtime/rest (the
// GET /api/data/process handler). Keeping it in a leaf package that imports
// only stdlib and libs/atlas-redis is what lets both runtime packages depend
// on it without an import cycle.
package ingestrun

import (
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// Namespace is the Redis namespace for every ingest/job-lifecycle key. The
// full key shape is <keyPrefix>:data-ingest:<suffix>, where <keyPrefix> comes
// from libs/atlas-redis KeyPrefix() (env-aware, so PR overlays are isolated).
const Namespace = "data-ingest"

// RunKeySuffix is appended to the per-run key suffix to address the run
// record, keeping it distinct from the job-name and heartbeat keys, which are
// typed Registry[string, string] and read by the Watchdog.
const RunKeySuffix = ":run"

// HeartbeatKeySuffix addresses the ingest pod's liveness timestamp.
const HeartbeatKeySuffix = ":updatedAt"

// RecordTTL bounds how long a run record survives. Refreshed on every write,
// so an in-flight run cannot expire mid-run. Redis is not a durable store: an
// eviction loses the record and the endpoint then reports PhaseNone, which is
// deliberately non-blocking for the baseline publish control.
const RecordTTL = 7 * 24 * time.Hour

// Phase is the overall state of an ingest run.
type Phase string

const (
	// PhaseNone means no run record exists for the triple.
	PhaseNone Phase = "none"
	// PhaseRunning means a run is in flight.
	PhaseRunning Phase = "running"
	// PhaseSucceeded means every worker finished without error.
	PhaseSucceeded Phase = "succeeded"
	// PhaseFailed means the ingest process returned an error.
	PhaseFailed Phase = "failed"
	// PhaseStuck means the Watchdog deleted the Job for heartbeat staleness.
	PhaseStuck Phase = "stuck"
	// PhaseUnknown is computed at read time: the record says running but
	// neither a fresh heartbeat nor a live Job corroborates it. Never stored.
	PhaseUnknown Phase = "unknown"
)

// WorkerState is the state of one registered ingest worker within a run.
type WorkerState string

const (
	WorkerPending   WorkerState = "pending"
	WorkerRunning   WorkerState = "running"
	WorkerSucceeded WorkerState = "succeeded"
	WorkerFailed    WorkerState = "failed"
	// WorkerSkipped is the ErrCategoryAbsent case: a category genuinely
	// missing from a monolithic Data.wz (v12 has no Quest). A skipped worker
	// does not make a run non-succeeded.
	WorkerSkipped WorkerState = "skipped"
)

// WorkerEntry is one worker's slot in a run record.
type WorkerEntry struct {
	Name       string      `json:"name"`
	State      WorkerState `json:"state"`
	StartedAt  *time.Time  `json:"startedAt,omitempty"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Record is the whole run: one Redis key, read with a single Get.
//
// It is a plain serialisable DTO rather than an immutable model with a
// Builder: it is mutated inside an optimistic-lock closure that may re-run
// many times, and it guards no domain invariant. Mutation is still confined —
// every transition is a With* method returning a modified copy, so the
// closure stays a pure function of its input.
type Record struct {
	RunId      string        `json:"runId"`
	JobName    string        `json:"jobName"`
	Scope      string        `json:"scope"`
	Region     string        `json:"region"`
	Version    string        `json:"version"`
	Tenant     string        `json:"tenant,omitempty"`
	Phase      Phase         `json:"phase"`
	StartedAt  time.Time     `json:"startedAt"`
	FinishedAt *time.Time    `json:"finishedAt,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	Workers    []WorkerEntry `json:"workers"`
}

// KeySuffix returns the per-run key suffix. The full Redis key is
// <prefix>:data-ingest:<suffix>[:run|:updatedAt].
//
// scope here is the RAW scope ("shared" or "tenants/<uuid>"), never the
// sanitized Kubernetes label form — see runtime/rest/jobs.go.
func KeySuffix(scope, region string, major, minor uint16) string {
	return fmt.Sprintf("%s:%s:%d.%d", scope, region, major, minor)
}

// NewJobRegistry returns the env-global Registry for the job-name and
// heartbeat keys. The keyFn is the identity so callers supply the full suffix.
func NewJobRegistry(rdb *goredis.Client) *redis.Registry[string, string] {
	return redis.NewRegistry[string, string](rdb, Namespace, func(s string) string { return s })
}

// NewRunRegistry returns the env-global Registry for run records. Values are
// JSON-marshalled by the Registry itself.
func NewRunRegistry(rdb *goredis.Client) *redis.Registry[string, Record] {
	return redis.NewRegistry[string, Record](rdb, Namespace, func(s string) string { return s })
}

// NewRecord seeds a fresh running record with every roster name pending.
func NewRecord(runId, jobName, scope, region, version, tenantId string, startedAt time.Time, workerNames []string) Record {
	ws := make([]WorkerEntry, 0, len(workerNames))
	for _, n := range workerNames {
		ws = append(ws, WorkerEntry{Name: n, State: WorkerPending})
	}
	return Record{
		RunId:     runId,
		JobName:   jobName,
		Scope:     scope,
		Region:    region,
		Version:   version,
		Tenant:    tenantId,
		Phase:     PhaseRunning,
		StartedAt: startedAt.UTC(),
		Workers:   ws,
	}
}

// copyWorkers returns a shallow copy of r with its own Workers backing array.
// Every With* method starts here so a retried optimistic-lock closure cannot
// corrupt the value it was handed.
func (r Record) copyWorkers() Record {
	out := r
	out.Workers = append([]WorkerEntry(nil), r.Workers...)
	return out
}

func (r Record) indexOf(name string) int {
	for i := range r.Workers {
		if r.Workers[i].Name == name {
			return i
		}
	}
	return -1
}

// WithRoster appends any name not already present, as pending. Existing
// entries — including already-terminal ones — are left untouched.
func (r Record) WithRoster(names []string) Record {
	out := r.copyWorkers()
	for _, n := range names {
		if out.indexOf(n) < 0 {
			out.Workers = append(out.Workers, WorkerEntry{Name: n, State: WorkerPending})
		}
	}
	return out
}

// WithWorkerRunning marks name running at `at`, appending it if the record's
// roster predates it (an older REST pod wrote the record).
func (r Record) WithWorkerRunning(name string, at time.Time) Record {
	out := r.copyWorkers()
	t := at.UTC()
	if i := out.indexOf(name); i >= 0 {
		out.Workers[i].State = WorkerRunning
		out.Workers[i].StartedAt = &t
		out.Workers[i].FinishedAt = nil
		out.Workers[i].Error = ""
		return out
	}
	out.Workers = append(out.Workers, WorkerEntry{Name: name, State: WorkerRunning, StartedAt: &t})
	return out
}

// WithWorkerTerminal marks name with a terminal state at `at`. errMsg is
// stored only for WorkerFailed callers; pass "" otherwise.
func (r Record) WithWorkerTerminal(name string, state WorkerState, at time.Time, errMsg string) Record {
	out := r.copyWorkers()
	t := at.UTC()
	if i := out.indexOf(name); i >= 0 {
		out.Workers[i].State = state
		out.Workers[i].FinishedAt = &t
		out.Workers[i].Error = errMsg
		return out
	}
	out.Workers = append(out.Workers, WorkerEntry{Name: name, State: state, FinishedAt: &t, Error: errMsg})
	return out
}

// WithPhase sets the run phase, stamping FinishedAt for terminal phases and
// leaving StartedAt alone (the REST pod owns it — see design Q2). A non-empty
// reason overwrites any previous one; "" preserves it.
func (r Record) WithPhase(p Phase, at time.Time, reason string) Record {
	out := r.copyWorkers()
	out.Phase = p
	if reason != "" {
		out.Reason = reason
	}
	if out.IsTerminal() {
		t := at.UTC()
		out.FinishedAt = &t
	}
	return out
}

// IsTerminal reports whether the phase will not change again without a new run.
func (r Record) IsTerminal() bool {
	switch r.Phase {
	case PhaseSucceeded, PhaseFailed, PhaseStuck:
		return true
	default:
		return false
	}
}

// CompleteCount is the number of workers that will not change again.
func (r Record) CompleteCount() int {
	n := 0
	for _, w := range r.Workers {
		switch w.State {
		case WorkerSucceeded, WorkerSkipped, WorkerFailed:
			n++
		}
	}
	return n
}
