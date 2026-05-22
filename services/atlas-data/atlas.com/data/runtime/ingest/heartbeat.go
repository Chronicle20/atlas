package ingest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// heartbeatInterval is how often the ingest pod refreshes its Redis
// :updatedAt key. Must be < Watchdog.TimeoutSecs by a comfortable margin.
const heartbeatInterval = 30 * time.Second

// heartbeatTTL bounds how long a stale heartbeat survives if the pod dies
// without cleanup. Long enough that a transient Redis blip on the writer
// side does not flag the Job as stuck.
const heartbeatTTL = time.Hour

// runHeartbeat ticks every heartbeatInterval and refreshes the Redis
// `<jobKey>:updatedAt` key the REST pod's Watchdog reads to decide whether a
// Job is stuck (see runtime/rest/watchdog.go:80-92, jobs.go:169-173).
//
// Without this refresher the heartbeat is written exactly once at Job creation
// and goes stale at TimeoutSecs, after which the Watchdog deletes the Job.
// PR-544 evidence: ingest pod created 01:23:30Z, last log 01:53:58Z, ~30 min
// match with the 1800s timeout — Map worker killed mid-execution, no
// `"map assets:"` summary emitted, ~80 maps including Henesys (100000000)
// left without layout.json/minimap.png in MinIO.
//
// Returns when ctx is cancelled. The first heartbeat fires immediately; we
// don't wait a full interval to refresh the timestamp the REST pod wrote.
func runHeartbeat(ctx context.Context, l logrus.FieldLogger, rdb *goredis.Client, key string) {
	if rdb == nil || key == "" {
		return
	}
	tick := func() {
		err := rdb.Set(ctx, key+":updatedAt", time.Now().UTC().Format(time.RFC3339), heartbeatTTL).Err()
		if err != nil && ctx.Err() == nil {
			l.WithError(err).Warnf("ingest heartbeat write failed (key=%s)", key)
		}
	}
	tick()
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		}
	}
}

// redisJobKeyFromEnv reconstructs the Watchdog's per-Job Redis key from the
// ingest pod's env vars. Shape matches runtime/rest/jobs.go:redisJobKey.
// Returns "" if any required env is missing so callers can skip heartbeating
// (e.g. unit-test / compose runs without the REST pod's key in Redis).
func redisJobKeyFromEnv() string {
	scope := os.Getenv("SCOPE")
	region := os.Getenv("REGION")
	if scope == "" || region == "" {
		return ""
	}
	major, err := strconv.ParseUint(os.Getenv("MAJOR_VERSION"), 10, 16)
	if err != nil {
		return ""
	}
	minor, err := strconv.ParseUint(os.Getenv("MINOR_VERSION"), 10, 16)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("atlas-data:ingest:%s:%s:%d.%d", scope, region, major, minor)
}
