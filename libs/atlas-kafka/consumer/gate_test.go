package consumer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
)

func TestGateProcessesTheLegacyEnvironment(t *testing.T) {
	// FR-1.8: with no records, everything is processed exactly as today.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id(""), false); got != gateProcess {
		t.Fatalf("verdict = %v, want gateProcess", got)
	}
}

func TestGateProcessesWhenThisDeploymentIsTheOwner(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateProcess {
		t.Fatalf("verdict = %v, want gateProcess (baseline owns a non-overridden service)", got)
	}
}

func TestGateSkipsWhenAnotherDeploymentIsTheOwner(t *testing.T) {
	// FR-4.4: acknowledged without domain processing.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{
		Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"}, Phase: env.PhaseActive,
	})

	if got := decide(r, env.Id("main"), "atlas-character", env.Id("pr-123"), false); got != gateSkipNotOwner {
		t.Fatalf("verdict = %v, want gateSkipNotOwner", got)
	}
}

func TestGateDropsAnUnknownEnvironment(t *testing.T) {
	// FR-4.7 / D4: not processed by ANY deployment, acked, alertable.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-999"), false); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestGateDropsAfterDeletion(t *testing.T) {
	// FR-5.7: after DELETED, surviving delayed work never executes against
	// the baseline. This is satisfied by the gate's drop path, not by
	// draining (design §7.4).
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	r.ApplyTombstone(env.Id("pr-123"))

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestGateDropsAMismatch(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("main"), true); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable for a header/tenant mismatch", got)
	}
}

func TestGateWhenStaleProcessesOnlyItsOwnEnvironment(t *testing.T) {
	// design §4.3: a registry outage degrades a baseline pod to exactly its
	// pre-project behaviour — it serves main and nothing else — rather than
	// taking it down.
	now := time.Unix(0, 0)
	r := env.NewMapRegistry(env.Id("main"), func() time.Time { return now })
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	now = now.Add(5 * time.Minute) // stale

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("main"), false); got != gateProcess {
		t.Fatalf("own environment while stale: verdict = %v, want gateProcess", got)
	}
	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateDropUnresolvable {
		t.Fatalf("foreign environment while stale: verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestExactlyOneDeploymentProcesses(t *testing.T) {
	// FR-4.6, stated as the property rather than as two separate asserts.
	overrides := map[string]string{"atlas-character": "atlas-pr-123"}
	records := []env.Record{
		{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive},
		{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Overrides: overrides, Phase: env.PhaseActive},
	}
	for _, svc := range []string{"atlas-character", "atlas-monsters"} {
		for _, msgEnv := range []env.Id{"main", "pr-123"} {
			processors := 0
			for _, self := range []env.Id{"main", "pr-123"} {
				r := env.NewMapRegistry(self, time.Now)
				for _, rec := range records {
					r.Apply(rec)
				}
				if decide(r, self, svc, msgEnv, false) == gateProcess {
					processors++
				}
			}
			if processors != 1 {
				t.Errorf("service=%s env=%s: %d deployments would process, want exactly 1", svc, msgEnv, processors)
			}
		}
	}
}

// TestGateDropUnresolvableIncrementsCounterAndSkipsHandler pins the two
// return-value traps: a drop must both increment the alertable counter AND
// leave the domain handler genuinely uninvoked, not merely a counter that
// moved while the handler still ran.
func TestGateDropUnresolvableIncrementsCounterAndSkipsHandler(t *testing.T) {
	env.SetRegistry(env.NewMapRegistry(env.Id("main"), time.Now))
	defer env.SetRegistry(nil)

	r := env.CurrentRegistry().(*env.MapRegistry)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})

	c := &Consumer{
		name:          "test-consumer",
		topic:         "test-topic",
		service:       "atlas-monsters",
		handlers:      make(map[string]handler.Handler),
		headerParsers: []HeaderParser{envHeaderParserForTest},
	}

	var handlerCalled bool
	var mu sync.Mutex
	c.handlers["h1"] = func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		mu.Lock()
		handlerCalled = true
		mu.Unlock()
		return true, nil
	}

	before := testutil.ToFloat64(gateDroppedUnresolvable.WithLabelValues("atlas-monsters", "pr-999"))

	msg := kafka.Message{
		Topic:   "test-topic",
		Headers: []kafka.Header{{Key: env.Key, Value: []byte("pr-999")}},
	}

	ok := c.processMessage(logrus.StandardLogger(), context.Background(), msg)
	if !ok {
		t.Fatalf("processMessage returned false on a drop; a drop must ack (return true) or it wedges the partition cursor")
	}

	mu.Lock()
	called := handlerCalled
	mu.Unlock()
	if called {
		t.Fatalf("domain handler was invoked for an unresolvable environment; the gate must run before any handler")
	}

	after := testutil.ToFloat64(gateDroppedUnresolvable.WithLabelValues("atlas-monsters", "pr-999"))
	if after != before+1 {
		t.Fatalf("atlas_kafka_gate_dropped_unresolvable_total = %v, want %v", after, before+1)
	}
}

// envHeaderParserForTest mirrors EnvHeaderParser but without a tenant
// dependency, so the counter/handler test above can drive the gate purely
// off the ENVIRONMENT header.
func envHeaderParserForTest(ctx context.Context, headers []kafka.Header) context.Context {
	var id env.Id
	for _, h := range headers {
		if h.Key == env.Key {
			id = env.Id(h.Value)
			break
		}
	}
	return env.WithContext(ctx, id)
}
