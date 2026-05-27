package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func newTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return c, mr
}

// TestRunHeartbeat_FirstTickIsImmediate proves runHeartbeat does not wait a
// full interval before writing — important when the REST pod's heartbeat is
// already approaching the Watchdog cutoff by the time the Job pod boots.
func TestRunHeartbeat_FirstTickIsImmediate(t *testing.T) {
	rdb, mr := newTestRedis(t)
	defer rdb.Close()
	l := logrus.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := "atlas-data:ingest:shared:GMS:83.1"
	go runHeartbeat(ctx, l, rdb, key)

	// Poll for the key with a tight budget — the first tick fires before the
	// ticker starts, so this should resolve well under one heartbeatInterval.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v, err := mr.Get(key + ":updatedAt"); err == nil && v != "" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected %s:updatedAt to be written within 2s of starting heartbeat", key)
}

// TestRunHeartbeat_NilClientNoop proves the function tolerates a nil Redis
// client (compose / test paths where SCOPE etc. are unset and no key is
// computed). The nil-key branch is exercised by ingest.Run; this covers the
// nil-client guard inside the goroutine.
func TestRunHeartbeat_NilClientNoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		runHeartbeat(ctx, logrus.New(), nil, "key")
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runHeartbeat did not return after ctx cancel")
	}
}

// TestRunHeartbeat_EmptyKeyNoop guards against the same shape but on the key
// argument — runtime/ingest/run.go skips spawning the goroutine when the key
// is empty; this is belt-and-braces if a caller ignores that contract.
func TestRunHeartbeat_EmptyKeyNoop(t *testing.T) {
	rdb, _ := newTestRedis(t)
	defer rdb.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		runHeartbeat(ctx, logrus.New(), rdb, "")
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runHeartbeat did not return after ctx cancel")
	}
}

// TestRedisJobKeyFromEnv_RoundTrips confirms the ingest-side key derivation
// matches the REST-side `redisJobKey` shape exactly. If these ever drift, the
// Watchdog reads a stale key and deletes live Jobs — silent in-prod failure.
// We assert the shape directly so a refactor of either side trips this test.
func TestRedisJobKeyFromEnv_RoundTrips(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "shared",
			env: map[string]string{
				"SCOPE":         "shared",
				"REGION":        "GMS",
				"MAJOR_VERSION": "83",
				"MINOR_VERSION": "1",
			},
			want: "atlas-data:ingest:shared:GMS:83.1",
		},
		{
			name: "tenant-scoped",
			env: map[string]string{
				"SCOPE":         "tenants/bf89e3b7-f154-4a2e-b8b7-2661614571a3",
				"REGION":        "GMS",
				"MAJOR_VERSION": "83",
				"MINOR_VERSION": "1",
			},
			want: "atlas-data:ingest:tenants/bf89e3b7-f154-4a2e-b8b7-2661614571a3:GMS:83.1",
		},
		{
			name: "missing scope returns empty",
			env: map[string]string{
				"REGION":        "GMS",
				"MAJOR_VERSION": "83",
				"MINOR_VERSION": "1",
			},
			want: "",
		},
		{
			name: "non-numeric major returns empty",
			env: map[string]string{
				"SCOPE":         "shared",
				"REGION":        "GMS",
				"MAJOR_VERSION": "x",
				"MINOR_VERSION": "1",
			},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("SCOPE", "")
			t.Setenv("REGION", "")
			t.Setenv("MAJOR_VERSION", "")
			t.Setenv("MINOR_VERSION", "")
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			if got := redisJobKeyFromEnv(); got != c.want {
				t.Errorf("redisJobKeyFromEnv() = %q, want %q", got, c.want)
			}
		})
	}
}
