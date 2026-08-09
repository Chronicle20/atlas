package rest

import (
	"atlas-data/ingestrun"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

func newTestRedis(t *testing.T) (*goredis.Client, *redis.Registry[string, string]) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return rdb, ingestrun.NewJobRegistry(rdb)
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestWatchdogSweep(t *testing.T) {
	now := time.Now().UTC()

	type job struct {
		name      string
		labels    map[string]string
		created   time.Time
		updatedAt *time.Time
		active    int32
		succeeded int32
	}

	cases := []struct {
		name        string
		timeoutSecs int
		jobs        []job
		wantPresent []string
		wantDeleted []string
	}{
		{
			name:        "deletes stuck job by redis updatedAt",
			timeoutSecs: 1800,
			jobs: []job{{
				name: "stuck",
				labels: map[string]string{
					labelIngest: "true",
					"scope":     "tenants-t", "region": "GMS", "version": "83.1", "tenant": "t",
				},
				created:   now.Add(-2 * time.Hour),
				updatedAt: ptrTime(now.Add(-1 * time.Hour)),
				active:    1,
			}},
			wantDeleted: []string{"stuck"},
		},
		{
			name:        "leaves healthy job alone",
			timeoutSecs: 1800,
			jobs: []job{{
				name: "healthy",
				labels: map[string]string{
					labelIngest: "true",
					"scope":     "tenants-t", "region": "GMS", "version": "83.1", "tenant": "t",
				},
				created:   now.Add(-10 * time.Minute),
				updatedAt: ptrTime(now),
				active:    1,
			}},
			wantPresent: []string{"healthy"},
		},
		{
			name:        "falls back to creation timestamp when no redis key",
			timeoutSecs: 1800,
			jobs: []job{{
				name: "old-no-heartbeat",
				labels: map[string]string{
					labelIngest: "true",
					"scope":     "tenants-t", "region": "GMS", "version": "83.1", "tenant": "t",
				},
				created: now.Add(-3 * time.Hour),
				active:  1,
			}},
			wantDeleted: []string{"old-no-heartbeat"},
		},
		{
			name:        "ignores succeeded jobs",
			timeoutSecs: 1800,
			jobs: []job{{
				name: "done",
				labels: map[string]string{
					labelIngest: "true",
					"scope":     "tenants-t", "region": "GMS", "version": "83.1", "tenant": "t",
				},
				created:   now.Add(-3 * time.Hour),
				succeeded: 1,
			}},
			wantPresent: []string{"done"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, reg := newTestRedis(t)
			objs := make([]runtime.Object, 0, len(tc.jobs))
			for _, jb := range tc.jobs {
				k8sJob := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:              jb.name,
						Namespace:         "ns",
						Labels:            jb.labels,
						CreationTimestamp: metav1.NewTime(jb.created),
					},
					Status: batchv1.JobStatus{Active: jb.active, Succeeded: jb.succeeded},
				}
				objs = append(objs, k8sJob)
				if jb.updatedAt != nil {
					suffix := ingestJobKeySuffixFromLabels(k8sJob)
					if suffix == "" {
						t.Fatalf("test setup: job %q missing scope/region/version labels", jb.name)
					}
					if err := reg.PutWithTTL(context.Background(), suffix+":updatedAt", jb.updatedAt.UTC().Format(time.RFC3339), time.Hour); err != nil {
						t.Fatal(err)
					}
				}
			}
			cs := fake.NewSimpleClientset(objs...)
			jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: reg}
			w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: tc.timeoutSecs}
			w.sweep(context.Background())

			for _, name := range tc.wantDeleted {
				if _, err := cs.BatchV1().Jobs("ns").Get(context.Background(), name, metav1.GetOptions{}); err == nil {
					t.Fatalf("expected job %q to be deleted", name)
				}
			}
			for _, name := range tc.wantPresent {
				if _, err := cs.BatchV1().Jobs("ns").Get(context.Background(), name, metav1.GetOptions{}); err != nil {
					t.Fatalf("expected job %q to remain present, got error: %v", name, err)
				}
			}
		})
	}
}

func TestWatchdogSweepNoK8sClient(t *testing.T) {
	w := Watchdog{L: logrus.New(), JobCreator: nil, TimeoutSecs: 60}
	w.sweep(context.Background()) // must not panic
}

func TestJobCreatorWritesHeartbeatToRedis(t *testing.T) {
	_, reg := newTestRedis(t)
	cs := fake.NewSimpleClientset()
	jc := &JobCreator{K8s: cs, Namespace: "ns", Template: testTemplate(), Registry: reg}
	name, err := jc.Create(context.Background(), "tenants/t1", "GMS", 83, 1, "t1", "")
	if err != nil {
		t.Fatal(err)
	}
	suffix := ingestrun.KeySuffix("tenants/t1", "GMS", 83, 1)
	got, err := reg.Get(context.Background(), suffix)
	if err != nil {
		t.Fatalf("registry missing job key suffix %q: %v", suffix, err)
	}
	if got != name {
		t.Fatalf("registry job key = %q, want %q", got, name)
	}
	if _, err := reg.Get(context.Background(), suffix+ingestrun.HeartbeatKeySuffix); err != nil {
		t.Fatalf("registry missing updatedAt: %v", err)
	}
}

func TestDeleteStuckJobWritesStuckRecord(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	scope := "tenants/t1"
	suffix := ingestrun.KeySuffix(scope, "GMS", 83, 1)

	// A record left mid-run: STRING done, MAP still going.
	rec := ingestrun.NewRecord("run-1", "j1", scope, "GMS", "83.1", "t1",
		time.Now().UTC().Add(-time.Hour), []string{"STRING", "MAP"})
	rec = rec.WithWorkerTerminal("STRING", ingestrun.WorkerSucceeded, time.Now().UTC(), "")
	rec = rec.WithWorkerRunning("MAP", time.Now().UTC())
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}
	if err := regs.Job.PutWithTTL(ctx, suffix, "j1", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := regs.Job.PutWithTTL(ctx, suffix+ingestrun.HeartbeatKeySuffix, time.Now().UTC().Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}

	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j1", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true", "scope": "tenants-t1",
			"region": "GMS", "version": "83.1", "tenant": "t1",
		},
	}}
	cs := fake.NewSimpleClientset(job)
	jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: regs.Job, RunRegistry: regs.Run}
	w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: 1800}

	w.deleteStuckJob(ctx, job)

	got, err := regs.Run.Get(ctx, suffix+ingestrun.RunKeySuffix)
	if err != nil {
		t.Fatalf("run record gone: %v", err)
	}
	if got.Phase != ingestrun.PhaseStuck {
		t.Fatalf("phase = %s, want stuck", got.Phase)
	}
	if got.Reason == "" {
		t.Fatal("stuck record has no reason")
	}
	if got.FinishedAt == nil {
		t.Fatal("stuck record has no finishedAt")
	}
	// Q1: the per-worker states are preserved exactly.
	if got.Workers[0].State != ingestrun.WorkerSucceeded {
		t.Fatalf("STRING = %s, want succeeded", got.Workers[0].State)
	}
	if got.Workers[1].State != ingestrun.WorkerRunning {
		t.Fatalf("MAP = %s, want running (preserved, not rewritten)", got.Workers[1].State)
	}
	// The two heartbeat keys are still removed, as before.
	if _, err := regs.Job.Get(ctx, suffix); err == nil {
		t.Fatal("job-name key not removed")
	}
	if _, err := regs.Job.Get(ctx, suffix+ingestrun.HeartbeatKeySuffix); err == nil {
		t.Fatal("heartbeat key not removed")
	}
}

func TestDeleteStuckJobDoesNotClobberNewerRun(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	scope := "tenants/t1"
	suffix := ingestrun.KeySuffix(scope, "GMS", 83, 1)

	// A newer run's live record — a re-triggered ingest for the same
	// (scope, region, version) that started after the stuck sweep began.
	newer := ingestrun.NewRecord("run-new", "j2", scope, "GMS", "83.1", "t1",
		time.Now().UTC(), []string{"STRING", "MAP"})
	newer = newer.WithWorkerRunning("STRING", time.Now().UTC())
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, newer, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	// The stuck Job belongs to the OLDER run (run-old), reconstructed here from
	// the same INGEST_RUN_ID env var the JobCreator stamps at creation time.
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "j1", Namespace: "ns",
			Labels: map[string]string{
				labelIngest: "true", "scope": "tenants-t1",
				"region": "GMS", "version": "83.1", "tenant": "t1",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "ingest",
						Env:  []corev1.EnvVar{{Name: "INGEST_RUN_ID", Value: "run-old"}},
					}},
				},
			},
		},
	}
	cs := fake.NewSimpleClientset(job)
	jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: regs.Job, RunRegistry: regs.Run}
	w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: 1800}

	w.deleteStuckJob(ctx, job)

	got, err := regs.Run.Get(ctx, suffix+ingestrun.RunKeySuffix)
	if err != nil {
		t.Fatalf("newer run record gone: %v", err)
	}
	if got.RunId != "run-new" {
		t.Fatalf("runId = %s, want run-new (record must not be overwritten)", got.RunId)
	}
	if got.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s, want running (stuck-old must not clobber newer run)", got.Phase)
	}
	if got.Reason != "" {
		t.Fatalf("reason = %q, want empty (untouched)", got.Reason)
	}
	if got.Workers[0].State != ingestrun.WorkerRunning {
		t.Fatalf("STRING = %s, want running (untouched)", got.Workers[0].State)
	}
}

func TestDeleteStuckJobWithNoRecordIsQuiet(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j1", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true", "scope": "shared", "region": "GMS", "version": "83.1",
		},
	}}
	cs := fake.NewSimpleClientset(job)
	jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: regs.Job, RunRegistry: regs.Run}
	w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: 1800}

	// Must not panic and must not resurrect a record that was never written.
	w.deleteStuckJob(ctx, job)

	suffix := ingestrun.KeySuffix("shared", "GMS", 83, 1)
	if _, err := regs.Run.Get(ctx, suffix+ingestrun.RunKeySuffix); err == nil {
		t.Fatal("deleteStuckJob created a record where none existed")
	}
}
