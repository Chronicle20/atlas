//go:build integration

package consumer_test

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

// The dwell harness models the live conditions behind the task-136 incident:
// a single broker, one consumer group with many idle topics plus one active
// topic (atlas-saga-orchestrator registers 15 consumers under one GroupID),
// and a second group sharing the coordinator. Each scenario measures
// publish→handler latency end-to-end: the publisher stamps send-time (unix
// nanos) into the message value and the handler records time.Since(stamp).
//
// S2/S4 assert POST-fix bounds (no self-recreate on idle, p99 < 1s). On
// pre-fix code they fail with the reproduced dwell — that failure is the
// baseline capture for findings.md, per the design (§4.1).

const dwellActiveTopic = "dwell.active"

func startDwellKafka(t *testing.T) []string {
	t.Helper()
	ctx := context.Background()
	kc, err := tckafka.Run(ctx, "confluentinc/cp-kafka:7.6.0", tckafka.WithClusterID("atlas-dwell"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = kc.Terminate(context.Background()) })
	brokers, err := kc.Brokers(ctx)
	require.NoError(t, err)
	return brokers
}

func createDwellTopics(t *testing.T, brokers []string, topics []string) {
	t.Helper()
	conn, err := (&kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}).
		DialContext(context.Background(), "tcp", brokers[0])
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	cfgs := make([]kafka.TopicConfig, 0, len(topics))
	for _, tp := range topics {
		cfgs = append(cfgs, kafka.TopicConfig{Topic: tp, NumPartitions: 1, ReplicationFactor: 1})
	}
	require.NoError(t, conn.CreateTopics(cfgs...))
}

func idleTopics(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("%s.idle.%d", prefix, i))
	}
	return out
}

type latencyRecorder struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (r *latencyRecorder) record(msg kafka.Message) {
	ns, err := strconv.ParseInt(string(msg.Value), 10, 64)
	if err != nil {
		return
	}
	d := time.Since(time.Unix(0, ns))
	r.mu.Lock()
	r.latencies = append(r.latencies, d)
	r.mu.Unlock()
}

func (r *latencyRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.latencies)
}

func (r *latencyRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latencies = nil
}

func (r *latencyRecorder) sorted() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := append([]time.Duration(nil), r.latencies...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s
}

func (r *latencyRecorder) p99() time.Duration {
	s := r.sorted()
	if len(s) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(s))*0.99)) - 1
	if idx < 0 {
		idx = 0
	}
	return s[idx]
}

func (r *latencyRecorder) max() time.Duration {
	s := r.sorted()
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}

// publishStamped writes n messages whose value is the send-time in unix
// nanoseconds, waiting interval between sends. WriteMessages blocks until
// the broker acks (RequireAll), so the stamp precedes broker persistence by
// at most the 10ms batch timeout.
func publishStamped(t *testing.T, brokers []string, topic string, n int, interval time.Duration) {
	t.Helper()
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}
	defer func() { _ = w.Close() }()
	for i := 0; i < n; i++ {
		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("k%d", i)),
			Value: []byte(strconv.FormatInt(time.Now().UnixNano(), 10)),
		}
		require.NoError(t, w.WriteMessages(context.Background(), msg))
		if interval > 0 {
			time.Sleep(interval)
		}
	}
}

func dumpSnapshots(t *testing.T, cm *consumer.Manager) {
	t.Helper()
	for _, c := range cm.Consumers() {
		t.Logf("snapshot: %+v", c.Snapshot())
	}
}

func totalRecreates(cm *consumer.Manager) int {
	total := 0
	for _, c := range cm.Consumers() {
		total += c.Snapshot().RecreateCount
	}
	return total
}

// dwellSetup boots the manager with the modeled topology and a latency
// recorder on the active topic. idleDecorators apply to the idle-topic
// consumers only (S2 uses them to compress the wedge cadence).
func dwellSetup(t *testing.T, brokers []string, idleCount int, otherCount int,
	rp []consumer.ManagerConfig, idleDecorators ...model.Decorator[consumer.Config],
) (*consumer.Manager, *latencyRecorder, context.CancelFunc, *sync.WaitGroup) {
	t.Helper()
	idle := idleTopics("dwell", idleCount)
	other := idleTopics("other", otherCount)
	all := append(append([]string{}, idle...), dwellActiveTopic)
	createDwellTopics(t, brokers, append(all, other...))

	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp...)
	add := cm.AddConsumer(l, ctx, wg)
	for _, tp := range idle {
		add(consumer.NewConfig(brokers, "dwell-"+tp, tp, "dwell-svc"), idleDecorators...)
	}
	add(consumer.NewConfig(brokers, "dwell-active", dwellActiveTopic, "dwell-svc"))
	for _, tp := range other {
		add(consumer.NewConfig(brokers, "other-"+tp, tp, "other-svc"))
	}

	rec := &latencyRecorder{}
	_, err := cm.RegisterHandler(dwellActiveTopic, func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		rec.record(msg)
		return true, nil
	})
	require.NoError(t, err)

	// Warm-up: group join for ~20 members can take tens of seconds; the
	// first delivery proves assignment settled and is excluded from
	// measurement.
	publishStamped(t, brokers, dwellActiveTopic, 1, 0)
	require.Eventually(t, func() bool { return rec.count() >= 1 },
		120*time.Second, 200*time.Millisecond, "warm-up message never delivered — group join did not settle")
	rec.reset()

	return cm, rec, cancel, wg
}

// S1 — steady state: full modeled fan-out, healthy consumers, production
// defaults. Asserts the PRD §8 target.
func TestDwellS1_SteadyStateLatency(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 15, 4, nil)
	defer func() { cancel(); wg.Wait() }()

	const n = 100
	publishStamped(t, brokers, dwellActiveTopic, n, 100*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= n },
		60*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S1: p99=%v max=%v over %d messages", p99, rec.max(), rec.count())
	dumpSnapshots(t, cm)
	require.Less(t, p99, time.Second, "S1: steady-state p99 publish→handler latency must be < 1s (PRD §8)")
}

// S2 — idle-tick churn (H1). Short fetchTimeout + low threshold on the 15
// idle consumers compresses the legacy 3×5m wedge cadence into seconds:
// pre-fix, every idle consumer self-wedges every ~4s, each Close() sends
// LeaveGroup and rebalances the whole group — including the active topic's
// member — reproducing the live dwell. Post-fix these deadlines are idle
// ticks: zero self-recreates, latency stays at the S1 bound.
func TestDwellS2_IdleTickChurn(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 15, 0, nil,
		consumer.SetFetchTimeout(2*time.Second),
		consumer.SetMaxConsecutiveTimeouts(2),
		// maxWait must stay well under fetchTimeout so an idle reader completes
		// >=1 fetch long-poll per deadline tick (its Fetches delta is the
		// idle-vs-stuck progress signal); mirrors the production 10s<<1m ratio.
		consumer.SetMaxWait(200*time.Millisecond),
	)
	defer func() { cancel(); wg.Wait() }()

	// The initial group-join for this ~16-member group takes several seconds;
	// with the compressed 2s fetchTimeout, an idle reader that is still
	// rebalancing during join makes no fetch progress across >1 tick and
	// self-wedges ONCE before the group settles. Production never hits this
	// (its 1m fetchTimeout dwarfs join time). Exclude that startup transient
	// the same way dwellSetup excludes warm-up latency: let recreates
	// stabilize post-join, then assert NO NEW recreate occurs in steady state.
	baselineRecreates := 0
	require.Eventually(t, func() bool {
		cur := totalRecreates(cm)
		stable := cur == baselineRecreates
		baselineRecreates = cur
		return stable
	}, 30*time.Second, time.Second, "S2: recreate count never stabilized after group join")

	const n = 30
	publishStamped(t, brokers, dwellActiveTopic, n, time.Second)
	require.Eventually(t, func() bool { return rec.count() >= n },
		120*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S2: p99=%v max=%v recreates=%d", p99, rec.max(), totalRecreates(cm))
	dumpSnapshots(t, cm)
	require.Equal(t, baselineRecreates, totalRecreates(cm),
		"S2: no reader may self-recreate on idle deadline ticks in steady state (design §3-A)")
	// Prove the deadline ticks actually fired and were classified idle —
	// otherwise a too-long fetchTimeout would make this test vacuous.
	tickedIdle := 0
	for _, c := range cm.Consumers() {
		if c.Snapshot().IdleTicks > 0 {
			tickedIdle++
		}
	}
	require.GreaterOrEqual(t, tickedIdle, 10,
		"S2: expected most idle consumers to have recorded idle ticks (fetchTimeout=2s over a 30s window)")
	require.Less(t, p99, time.Second, "S2: churn-free p99 must be < 1s (PRD §8)")
}

// forceErrReader wraps a real reader; the test arms a one-shot injected
// fetch error to drive the genuine recreate path (close → LeaveGroup →
// backoff → recreate → rejoin) on the active topic mid-stream.
type forceErrReader struct {
	inner consumer.KafkaReader
	arm   *atomic.Bool
}

func (r *forceErrReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r.arm.CompareAndSwap(true, false) {
		return kafka.Message{}, fmt.Errorf("injected fetch failure")
	}
	return r.inner.FetchMessage(ctx)
}

func (r *forceErrReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return r.inner.CommitMessages(ctx, msgs...)
}

func (r *forceErrReader) Close() error { return r.inner.Close() }

// Stats delegates so idle-vs-stuck classification sees the real reader's
// progress. The anonymous interface avoids depending on the lib's
// StatsProvider name.
func (r *forceErrReader) Stats() kafka.ReaderStats {
	if sp, ok := r.inner.(interface{ Stats() kafka.ReaderStats }); ok {
		return sp.Stats()
	}
	return kafka.ReaderStats{}
}

// S3 — bounded recreate: a genuine failure on the active topic's reader
// forces one recreate mid-stream; delivery must resume with the recreate
// dwell bounded by the join+backoff budget (design §4.1: ≤ 10s).
func TestDwellS3_ForcedRecreateBounded(t *testing.T) {
	brokers := startDwellKafka(t)
	var arm atomic.Bool
	rp := consumer.ConfigReaderProducer(func(cfg kafka.ReaderConfig) consumer.KafkaReader {
		inner := kafka.NewReader(cfg)
		if cfg.Topic == dwellActiveTopic {
			return &forceErrReader{inner: inner, arm: &arm}
		}
		return inner
	})
	cm, rec, cancel, wg := dwellSetup(t, brokers, 5, 0, []consumer.ManagerConfig{rp})
	defer func() { cancel(); wg.Wait() }()

	// Healthy stretch first.
	publishStamped(t, brokers, dwellActiveTopic, 10, 100*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= 10 },
		30*time.Second, 100*time.Millisecond, "pre-recreate messages not delivered")
	rec.reset()

	// Inject the failure, then keep publishing across the recreate window.
	arm.Store(true)
	publishStamped(t, brokers, dwellActiveTopic, 20, 250*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= 20 },
		60*time.Second, 100*time.Millisecond, "messages lost across recreate")

	maxDwell := rec.max()
	active := snapshotForTopic(t, cm, dwellActiveTopic)
	t.Logf("S3: max dwell across recreate=%v recreates=%d timeToFirstFetch=%v totalBackoff=%v",
		maxDwell, active.RecreateCount, active.TimeToFirstFetch, active.TotalBackoff)
	dumpSnapshots(t, cm)
	require.GreaterOrEqual(t, active.RecreateCount, 1, "S3: injected error must have forced a recreate")
	require.LessOrEqual(t, maxDwell, 10*time.Second, "S3: recreate dwell must be bounded (design §4.1)")
}

// S4 — control (H3): deadline ticks at 2s cadence on a small healthy group.
// Post-fix, ticks alone add no dwell and cause no recreates — closing PRD
// §9 Q2 with measurement rather than only the kafka-go source citation.
func TestDwellS4_TickControl(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 2, 0, nil,
		consumer.SetFetchTimeout(2*time.Second),
		// maxWait must stay well under fetchTimeout so an idle reader completes
		// >=1 fetch long-poll per deadline tick (its Fetches delta is the
		// idle-vs-stuck progress signal); mirrors the production 10s<<1m ratio.
		consumer.SetMaxWait(200*time.Millisecond),
	)
	defer func() { cancel(); wg.Wait() }()

	const n = 30
	publishStamped(t, brokers, dwellActiveTopic, n, 500*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= n },
		60*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S4: p99=%v max=%v recreates=%d", p99, rec.max(), totalRecreates(cm))
	dumpSnapshots(t, cm)
	require.Zero(t, totalRecreates(cm), "S4: ticks alone must not recreate")
	require.Less(t, p99, time.Second, "S4: ticks alone must add no dwell")
}

// S5 — MaxWait A/B (H2): an idle group reader at maxWait=50ms vs 10s. With
// MinBytes=1 the broker answers immediately when data exists, so MaxWait
// only bounds the empty long-poll — the 50ms setting buys no latency and
// multiplies idle fetch traffic. Raw kafka.Readers are used (not the lib)
// so the test may consume Stats() deltas itself; findings.md extrapolates
// the per-reader rates to the live ~481 partitions.
func TestDwellS5_MaxWaitIdleFetchRate(t *testing.T) {
	brokers := startDwellKafka(t)
	createDwellTopics(t, brokers, []string{"s5.idle.a", "s5.idle.b"})

	measure := func(topic string, maxWait time.Duration) int64 {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: "s5-" + topic,
			MaxWait: maxWait,
		})
		defer func() { _ = r.Close() }()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Drive the reader; FetchMessage blocks the full window on an idle
		// topic while the background loop long-polls the broker.
		_, _ = r.FetchMessage(ctx)
		return r.Stats().Fetches
	}

	fast := measure("s5.idle.a", 50*time.Millisecond)
	slow := measure("s5.idle.b", 10*time.Second)
	t.Logf("S5: idle fetch attempts in 30s — maxWait=50ms: %d; maxWait=10s: %d", fast, slow)
	require.Greater(t, fast, slow, "S5: shorter MaxWait must issue more idle fetch requests")
}
