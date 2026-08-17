package ingest

import (
	"atlas-data/ingestrun"
	"context"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// heartbeatInterval is how often the ingest pod refreshes its Redis
// :updatedAt key. Must be < Watchdog.TimeoutSecs by a comfortable margin.
const heartbeatInterval = 30 * time.Second

// heartbeatTTL bounds how long a stale heartbeat survives if the pod dies
// without cleanup. Long enough that a transient Redis blip on the writer
// side does not flag the Job as stuck.
const heartbeatTTL = time.Hour

// runHeartbeat ticks every heartbeatInterval and refreshes the Redis
// `<suffix>:updatedAt` key the REST pod's Watchdog reads to decide whether a
// Job is stuck (see runtime/rest/watchdog.go:jobIsStuck, jobs.go:Create).
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
func runHeartbeat(ctx context.Context, l logrus.FieldLogger, reg *redis.EnvironmentRegistry[string, string], suffix string) {
	if reg == nil || suffix == "" {
		return
	}
	tick := func() {
		err := reg.PutWithTTL(ctx, env.Self(), suffix+":updatedAt", time.Now().UTC().Format(time.RFC3339), heartbeatTTL)
		if err != nil && ctx.Err() == nil {
			l.WithError(err).Warnf("ingest heartbeat write failed (suffix=%s)", suffix)
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

// ingestJobSuffixFromEnv reconstructs the Watchdog's per-Job key suffix from
// the ingest pod's env vars, via ingestrun.KeySuffix — the single producer of
// this namespace's key suffix (see tools/rediskeyguard's bareConstructorAllowlist
// entry for ingestrun.NewJobRegistry/NewRunRegistry: every derivation of a key
// in this namespace must go through KeySuffix or ingestJobKeySuffixFromLabels
// for that allowlist entry to stay sound).
// Returns "" if any required env is missing so callers can skip heartbeating
// (e.g. unit-test / compose runs without the REST pod's key in Redis).
func ingestJobSuffixFromEnv() string {
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
	return ingestrun.KeySuffix(scope, region, uint16(major), uint16(minor))
}

// ingestRunIdFromEnv returns the run identity JobCreator injected into the
// rendered Job. Empty in the compose / unit-test path, which disables the
// superseded-pod guard (there is no competing pod there).
func ingestRunIdFromEnv() string {
	return os.Getenv("INGEST_RUN_ID")
}
