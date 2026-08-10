package consumer

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// TestResolveEngineDefaultsToConsumerGroup: unset means the new engine
// (FR-5.1). The default is what ships; `reader` is the rollback.
func TestResolveEngineDefaultsToConsumerGroup(t *testing.T) {
	l, _ := test.NewNullLogger()
	t.Setenv(engineEnvVar, "")
	if got := resolveEngine(l); got != EngineConsumerGroup {
		t.Fatalf("resolveEngine = %q, want %q", got, EngineConsumerGroup)
	}
}

// TestResolveEngineHonoursReader is the rollback path: one env var, one pod
// restart, no state migration (FR-5.2).
func TestResolveEngineHonoursReader(t *testing.T) {
	l, _ := test.NewNullLogger()
	t.Setenv(engineEnvVar, "reader")
	if got := resolveEngine(l); got != EngineReader {
		t.Fatalf("resolveEngine = %q, want %q", got, EngineReader)
	}
}

// TestResolveEngineFailsSoftOnGarbage: a typo in a deployment env var must
// not take a service's consumers offline. Warn and use the default.
func TestResolveEngineFailsSoftOnGarbage(t *testing.T) {
	l, hook := test.NewNullLogger()
	t.Setenv(engineEnvVar, "readr")
	if got := resolveEngine(l); got != EngineConsumerGroup {
		t.Fatalf("resolveEngine = %q, want the default %q", got, EngineConsumerGroup)
	}
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level.String() == "warning" {
			found = true
		}
	}
	if !found {
		t.Fatal("no warning logged for an unrecognised engine value")
	}
}

// TestConfigEngineOverridesEnv: the explicit configurator wins, which is how
// the legacy-engine tests pin themselves without racing on process env.
func TestConfigEngineOverridesEnv(t *testing.T) {
	ResetInstance()
	t.Setenv(engineEnvVar, "consumergroup")
	m := GetManager(ConfigEngine(EngineReader))
	if m.engine != EngineReader {
		t.Fatalf("Manager.engine = %q, want %q", m.engine, EngineReader)
	}
	ResetInstance()
}

// TestSnapshotReportsEngine: during a staged rollout both engines run in one
// cluster, so "which engine is this pod on?" must be answerable from the
// debug route rather than from the pod's env (design §7).
func TestSnapshotReportsEngine(t *testing.T) {
	c := newTestConsumer()
	c.engine = EngineConsumerGroup
	if got := c.Snapshot().Engine; got != "consumergroup" {
		t.Fatalf("Snapshot.Engine = %q, want consumergroup", got)
	}
}
