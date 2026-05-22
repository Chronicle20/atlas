package rest

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
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
					"scope":     "tenants-t", "region": "GMS", "version": "83.1",
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
					"scope":     "tenants-t", "region": "GMS", "version": "83.1",
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
					"scope":     "tenants-t", "region": "GMS", "version": "83.1",
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
					"scope":     "tenants-t", "region": "GMS", "version": "83.1",
				},
				created:   now.Add(-3 * time.Hour),
				succeeded: 1,
			}},
			wantPresent: []string{"done"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := newTestRedis(t)
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
					key := redisJobKeyFromLabels(k8sJob)
					if key == "" {
						t.Fatalf("test setup: job %q missing scope/region/version labels", jb.name)
					}
					if err := rdb.Set(context.Background(), key+":updatedAt", jb.updatedAt.UTC().Format(time.RFC3339), time.Hour).Err(); err != nil {
						t.Fatal(err)
					}
				}
			}
			cs := fake.NewSimpleClientset(objs...)
			jc := &JobCreator{K8s: cs, Namespace: "ns", Redis: rdb}
			w := Watchdog{L: logrus.New(), JobCreator: jc, Redis: rdb, TimeoutSecs: tc.timeoutSecs}
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
	rdb := newTestRedis(t)
	cs := fake.NewSimpleClientset()
	jc := &JobCreator{K8s: cs, Namespace: "ns", Template: testTemplate(), Redis: rdb}
	name, err := jc.Create(context.Background(), "tenants/t1", "GMS", 83, 1, "t1", "")
	if err != nil {
		t.Fatal(err)
	}
	key := redisJobKey("tenants/t1", "GMS", 83, 1)
	got, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("redis missing job key %q: %v", key, err)
	}
	if got != name {
		t.Fatalf("redis job key = %q, want %q", got, name)
	}
	if _, err := rdb.Get(context.Background(), key+":updatedAt").Result(); err != nil {
		t.Fatalf("redis missing updatedAt: %v", err)
	}
}
