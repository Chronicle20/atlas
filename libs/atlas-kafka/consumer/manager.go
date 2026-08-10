package consumer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

type KafkaReader interface {
	MessageReader
	MessageCommitter
	Closer
}

// Closer is a subset of io.Closer — defined locally so we don't have to import
// io solely for one interface.
type Closer interface {
	Close() error
}

type MessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
}

type MessageCommitter interface {
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

// StatsProvider is implemented by readers that can report kafka-go reader
// statistics; *kafka.Reader satisfies it natively. The fetch loop uses
// Stats() deltas to distinguish an idle reader (still issuing fetch
// attempts against the broker) from a stuck one (no progress at all).
//
// OWNERSHIP: kafka-go's Stats() returns counter deltas since the previous
// call. This lib owns the reader's stats stream exclusively — nothing else
// may call Stats() on a lib-owned reader, or both callers see partial
// deltas. External metrics/telemetry must read Consumer.Snapshot() instead.
type StatsProvider interface {
	Stats() kafka.ReaderStats
}

// readerMadeProgress reports whether the reader has done any work since the
// previous deadline tick. Readers that don't expose Stats() (test mocks)
// are conservatively treated as making no progress — legacy behavior, where
// every deadline tick counts toward the wedge threshold.
func readerMadeProgress(reader KafkaReader) bool {
	sp, ok := reader.(StatsProvider)
	if !ok {
		return false
	}
	s := sp.Stats()
	return s.Fetches > 0 || s.Dials > 0 || s.Messages > 0
}

type ReaderProducer func(config kafka.ReaderConfig) KafkaReader

type ManagerConfig func(m *Manager)

//goland:noinspection GoUnusedExportedFunction
func ConfigReaderProducer(rp ReaderProducer) ManagerConfig {
	return func(m *Manager) {
		m.rp = rp
	}
}

type Manager struct {
	mu        *sync.Mutex
	consumers map[string]*Consumer
	rp        ReaderProducer
	gp        GroupProducer
	prp       PartitionReaderProducer
}

var (
	manager *Manager
	once    sync.Once
)

func ResetInstance() {
	manager = nil
	once = sync.Once{}
}

//goland:noinspection GoUnusedExportedFunction
func GetManager(configurators ...ManagerConfig) *Manager {
	once.Do(func() {
		manager = &Manager{
			mu:        &sync.Mutex{},
			consumers: make(map[string]*Consumer),
			rp: func(config kafka.ReaderConfig) KafkaReader {
				return kafka.NewReader(config)
			},
			gp:  defaultGroupProducer,
			prp: defaultPartitionReaderProducer,
		}
		for _, configurator := range configurators {
			configurator(manager)
		}
	})
	return manager
}

// Consumers returns a snapshot of all registered consumers. Ordering is
// unspecified. Callers must not mutate the returned slice or its contents —
// it is safe for read-only inspection (e.g., debug routes).
func (m *Manager) Consumers() []*Consumer {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Consumer, 0, len(m.consumers))
	for _, c := range m.consumers {
		out = append(out, c)
	}
	return out
}

func (m *Manager) AddConsumer(cl logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config Config, decorators ...model.Decorator[Config]) {
	return func(config Config, decorators ...model.Decorator[Config]) {
		m.mu.Lock()
		defer m.mu.Unlock()

		c := config
		for _, d := range decorators {
			c = d(c)
		}

		if _, exists := m.consumers[c.topic]; exists {
			cl.Infof("Consumer for topic [%s] is already registered.", c.topic)
			return
		}

		// Guard the idle-vs-stuck classification invariant (task-136 — see
		// docs/tasks/task-136-consumer-fetch-wedge/findings.md): an idle
		// reader's Stats().Fetches increments roughly once per maxWait
		// interval, so handleFetchDeadline only sees progress if
		// fetchTimeout is comfortably greater than maxWait. A misconfigured
		// consumer (maxWait >= fetchTimeout) can complete zero fetches per
		// liveness tick, get misclassified as no-progress, and be wrongly
		// recreated. This is a one-time Warn at registration — never a
		// clamp — so the misconfiguration is visible without changing
		// behavior.
		if c.maxWait >= c.fetchTimeout {
			cl.Warnf("Consumer for topic [%s] (group [%s]) has maxWait (%v) >= fetchTimeout (%v); an idle reader may not complete a fetch per liveness tick and could be wrongly recreated. Set fetchTimeout comfortably above maxWait.",
				c.topic, c.groupId, c.maxWait, c.fetchTimeout)
		}

		readerConfig := kafka.ReaderConfig{
			Brokers:     c.brokers,
			Topic:       c.topic,
			GroupID:     c.groupId,
			MaxWait:     c.maxWait,
			StartOffset: c.startOffset,
		}

		maxInFlight := c.maxInFlight
		if maxInFlight < 1 {
			maxInFlight = 1
		}
		con := &Consumer{
			name:                   c.name,
			topic:                  c.topic,
			groupId:                c.groupId,
			brokers:                append([]string(nil), c.brokers...),
			readerConfig:           readerConfig,
			rp:                     m.rp,
			handlers:               make(map[string]handler.Handler),
			headerParsers:          c.headerParsers,
			fetchTimeout:           c.fetchTimeout,
			maxConsecutiveTimeouts: c.maxConsecutiveTimeouts,
			maxInFlight:            maxInFlight,
			maxWait:                c.maxWait,
			startOffset:            c.startOffset,
			gp:                     m.gp,
			prp:                    m.prp,
		}

		m.consumers[c.topic] = con

		l := cl.WithFields(logrus.Fields{"originator": c.topic, "type": "kafka_consumer"})
		routine.Go(l, ctx, func(_ context.Context) { con.start(l, ctx, wg) })
	}
}

func (m *Manager) RegisterHandler(topic string, handler handler.Handler) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	consumer, exists := m.consumers[topic]
	if !exists {
		return "", errors.New("no consumer found for topic")
	}

	handlerId := uuid.New().String()
	consumer.mu.Lock()
	consumer.handlers[handlerId] = handler
	consumer.mu.Unlock()

	return handlerId, nil
}

func (m *Manager) AddConsumerAndRegister(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(c Config, h handler.Handler) (string, error) {
	return func(c Config, h handler.Handler) (string, error) {
		m.AddConsumer(l, ctx, wg)(c)
		return m.RegisterHandler(c.topic, h)
	}
}

func (m *Manager) RemoveHandler(topic string, handlerId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	consumer, exists := m.consumers[topic]
	if !exists {
		return errors.New("no consumer found for topic")
	}

	consumer.mu.Lock()
	delete(consumer.handlers, handlerId)
	consumer.mu.Unlock()
	return nil
}

// Consumer owns a single Kafka reader for a single topic. Its reader is
// allowed to die (EOF, retry exhaustion, rebalance errors) — the outer
// lifecycle loop in start rebuilds the reader and rejoins the consumer
// group without disturbing the surrounding process.
type Consumer struct {
	name          string
	topic         string
	groupId       string
	brokers       []string
	readerConfig  kafka.ReaderConfig
	rp            ReaderProducer
	gp            GroupProducer
	prp           PartitionReaderProducer
	handlers      map[string]handler.Handler
	headerParsers []HeaderParser
	mu            sync.Mutex

	// Read-only after construction; copied from Config in AddConsumer.
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int
	maxInFlight            int
	maxWait                time.Duration
	startOffset            int64

	// Observable state — protected by mu.
	aliveSince    time.Time
	lastErrorAt   time.Time
	lastError     string
	recreateCount int

	// Watchdog counters live per assigned partition. The legacy engine has
	// no partition of its own and uses the legacyPartition key, so its
	// single-entry map aggregates in Snapshot to exactly the scalars this
	// replaced (task-209 design §7).
	partitions map[int]*partitionState

	// Assignment state — meaningful on the consumergroup engine only.
	assignedPartitions []int
	generationID       int32
	lastAssignmentAt   time.Time

	// Phase-timing attribution — protected by mu. Durations are monotonic
	// deltas around existing call sites; they exist so a dwell can be
	// attributed to a phase (fetch wait, group join, recreate backoff,
	// handler dispatch) via Snapshot without a profiler.
	readerCreatedAt     time.Time
	awaitingFirstFetch  bool
	timeToFirstFetch    time.Duration
	lastFetchDuration   time.Duration
	maxFetchDuration    time.Duration
	lastHandlerDuration time.Duration
	maxHandlerDuration  time.Duration
	totalBackoff        time.Duration
}

// legacyPartition is the partitionState key used by the legacy *kafka.Reader
// engine, which never sees a partition id of its own. Real partition ids are
// non-negative, so -1 can never collide.
const legacyPartition = -1

// partitionState holds the liveness-watchdog counters for one partition. A
// scalar would let a healthy partition's resets mask a sibling partition's
// wedge; keying by partition is why the wedge stays detectable on a
// multi-partition topic.
type partitionState struct {
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
	idleTicks           int
	lastIdleTickAt      time.Time
	noProgressTicks     int
	lastNoProgressAt    time.Time
	lastFetchAt         time.Time
	backoff             *fetchBackoff
}

func newPartitionState() *partitionState {
	return &partitionState{backoff: newFetchBackoff()}
}

// partitionStateFor returns the state for partition, creating it on first use.
func (c *Consumer) partitionStateFor(partition int) *partitionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.partitionStateLocked(partition)
}

func (c *Consumer) partitionStateLocked(partition int) *partitionState {
	if c.partitions == nil {
		c.partitions = make(map[int]*partitionState)
	}
	st, ok := c.partitions[partition]
	if !ok {
		st = newPartitionState()
		c.partitions[partition] = st
	}
	return st
}

// onAssignment records the generation's assignment for this consumer's topic.
// State for partitions the consumer no longer holds is dropped, so a
// partition that comes back after a gap starts with clean no-progress
// counters (FR-2.4); state for partitions retained across the generation is
// preserved so operators keep the accumulated tick history.
func (c *Consumer) onAssignment(genID int32, partitions []int) {
	sorted := append([]int(nil), partitions...)
	sort.Ints(sorted)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.generationID = genID
	c.lastAssignmentAt = time.Now()
	c.assignedPartitions = sorted

	next := make(map[int]*partitionState, len(sorted))
	for _, p := range sorted {
		if st, ok := c.partitions[p]; ok {
			next[p] = st
		} else {
			next[p] = newPartitionState()
		}
	}
	c.partitions = next
}

// Snapshot is a point-in-time view of a Consumer's observable state, suitable
// for JSON serialization by the debug route.
type Snapshot struct {
	Name        string
	Topic       string
	GroupID     string
	Brokers     []string
	AliveSince  time.Time
	LastFetchAt time.Time
	LastErrorAt time.Time
	LastError   string
	// RecreateCount counts reader rebuilds. On the legacy engine each rebuild
	// is a consumer-group REJOIN (it rebalances every member of the group);
	// on the consumergroup engine it is a local partition-reader rebuild with
	// no broker-visible group effect. Do not compare the number across a
	// KAFKA_CONSUMER_ENGINE rollback.
	RecreateCount       int
	HandlerCount        int
	LastTimeoutAt       time.Time
	ConsecutiveTimeouts int
	IdleTicks           int
	LastIdleTickAt      time.Time
	NoProgressTicks     int
	LastNoProgressAt    time.Time
	// AssignedPartitions is the sorted partition list this consumer holds in
	// the current generation. Always non-nil; empty means healthy-idle (or
	// the legacy engine, which does not observe assignment).
	AssignedPartitions  []int
	GenerationID        int32
	LastAssignmentAt    time.Time
	TimeToFirstFetch    time.Duration
	LastFetchDuration   time.Duration
	MaxFetchDuration    time.Duration
	LastHandlerDuration time.Duration
	MaxHandlerDuration  time.Duration
	TotalBackoff        time.Duration
}

// Snapshot returns a consistent snapshot of the consumer's observable state.
func (c *Consumer) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		consecutive      int
		idleTicks        int
		noProgressTicks  int
		lastTimeoutAt    time.Time
		lastIdleTickAt   time.Time
		lastNoProgressAt time.Time
		lastFetchAt      time.Time
	)
	latest := func(dst *time.Time, v time.Time) {
		if v.After(*dst) {
			*dst = v
		}
	}
	for _, st := range c.partitions {
		if st.consecutiveTimeouts > consecutive {
			consecutive = st.consecutiveTimeouts
		}
		idleTicks += st.idleTicks
		noProgressTicks += st.noProgressTicks
		latest(&lastTimeoutAt, st.lastTimeoutAt)
		latest(&lastIdleTickAt, st.lastIdleTickAt)
		latest(&lastNoProgressAt, st.lastNoProgressAt)
		latest(&lastFetchAt, st.lastFetchAt)
	}

	return Snapshot{
		Name:                c.name,
		Topic:               c.topic,
		GroupID:             c.groupId,
		Brokers:             append([]string(nil), c.brokers...),
		AliveSince:          c.aliveSince,
		LastFetchAt:         lastFetchAt,
		LastErrorAt:         c.lastErrorAt,
		LastError:           c.lastError,
		RecreateCount:       c.recreateCount,
		HandlerCount:        len(c.handlers),
		LastTimeoutAt:       lastTimeoutAt,
		ConsecutiveTimeouts: consecutive,
		IdleTicks:           idleTicks,
		LastIdleTickAt:      lastIdleTickAt,
		NoProgressTicks:     noProgressTicks,
		LastNoProgressAt:    lastNoProgressAt,
		AssignedPartitions:  append([]int{}, c.assignedPartitions...),
		GenerationID:        c.generationID,
		LastAssignmentAt:    c.lastAssignmentAt,
		TimeToFirstFetch:    c.timeToFirstFetch,
		LastFetchDuration:   c.lastFetchDuration,
		MaxFetchDuration:    c.maxFetchDuration,
		LastHandlerDuration: c.lastHandlerDuration,
		MaxHandlerDuration:  c.maxHandlerDuration,
		TotalBackoff:        c.totalBackoff,
	}
}

func (c *Consumer) onReaderCreated(partition int, attempt int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.aliveSince = now
	c.readerCreatedAt = now
	c.awaitingFirstFetch = true
	if attempt > 0 {
		c.recreateCount++
		c.lastError = ""
		st := c.partitionStateLocked(partition)
		st.consecutiveTimeouts = 0
		st.lastTimeoutAt = time.Time{}
	}
}

func (c *Consumer) recordFetch(partition int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastError = ""
	st := c.partitionStateLocked(partition)
	st.lastFetchAt = now
	st.consecutiveTimeouts = 0
	if c.awaitingFirstFetch {
		c.timeToFirstFetch = now.Sub(c.readerCreatedAt)
		c.awaitingFirstFetch = false
	}
}

// recordIdleTick marks one deadline expiration on a reader that is still
// making fetch attempts. Idle is healthy: it resets the no-progress
// escalation counter and touches no error state.
func (c *Consumer) recordIdleTick(partition int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.partitionStateLocked(partition)
	st.idleTicks++
	st.lastIdleTickAt = time.Now()
	st.consecutiveTimeouts = 0
}

// recordNoProgressTick marks one deadline expiration with zero reader
// progress — a stall suspect. Returns the new consecutive count so callers
// can branch on the threshold without a second mutex acquisition.
func (c *Consumer) recordNoProgressTick(partition int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	st := c.partitionStateLocked(partition)
	st.lastTimeoutAt = now
	st.lastNoProgressAt = now
	st.noProgressTicks++
	st.consecutiveTimeouts++
	return st.consecutiveTimeouts
}

func (c *Consumer) recordError(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastErrorAt = time.Now()
	c.lastError = err.Error()
}

func (c *Consumer) recordFetchDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFetchDuration = d
	if d > c.maxFetchDuration {
		c.maxFetchDuration = d
	}
}

func (c *Consumer) recordHandlerDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastHandlerDuration = d
	if d > c.maxHandlerDuration {
		c.maxHandlerDuration = d
	}
}

func (c *Consumer) recordBackoff(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalBackoff += d
}

// handleFetchDeadline classifies one expired fetch deadline: an idle tick
// (reader made progress — normal on a no-traffic topic) or a no-progress
// tick (stall suspect). Returns errFetchWedged once consecutive no-progress
// ticks reach the threshold, nil otherwise.
//
// On the consumergroup engine this is only ever reached from a reader that
// HOLDS a partition assignment, which is what makes FR-2.5 structural: an
// unassigned consumer has no partition loop and so cannot emit these warns.
func (c *Consumer) handleFetchDeadline(l logrus.FieldLogger, reader KafkaReader, partition int) error {
	if readerMadeProgress(reader) {
		c.recordIdleTick(partition)
		l.Debugf("Fetch deadline expired on idle topic [%s]; reader healthy, continuing.", c.topic)
		return nil
	}
	consecutive := c.recordNoProgressTick(partition)
	if consecutive >= c.maxConsecutiveTimeouts {
		l.Warnf("FetchMessage wedged: %d consecutive no-progress ticks on topic [%s] (group [%s]); forcing reader recreate.",
			consecutive, c.topic, c.groupId)
		return errFetchWedged
	}
	l.Warnf("FetchMessage made no progress on topic [%s] (group [%s]) (consecutive=%d/%d); stall suspect.",
		c.topic, c.groupId, consecutive, c.maxConsecutiveTimeouts)
	return nil
}

// fetchBackoff models the outer reader-recreate backoff. Capped exponential
// starting at 500ms and doubling up to 10s. Reset to the initial on a
// successful fetch (handled by the outer loop via newFetchBackoff).
type fetchBackoff struct {
	current time.Duration
}

func newFetchBackoff() *fetchBackoff {
	return &fetchBackoff{}
}

func (b *fetchBackoff) next() time.Duration {
	const (
		initial = 500 * time.Millisecond
		cap_    = 10 * time.Second
	)
	if b.current == 0 {
		b.current = initial
		return b.current
	}
	b.current *= 2
	if b.current > cap_ {
		b.current = cap_
	}
	return b.current
}

// processMessage runs all handlers synchronously and returns true if all succeeded.
func (c *Consumer) processMessage(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) bool {
	wctx := ctx
	for _, p := range c.headerParsers {
		wctx = p(wctx, msg.Headers)
	}

	var span trace.Span
	wctx, span = otel.GetTracerProvider().Tracer("atlas-kafka").Start(wctx, c.name)
	handlerLogger := l.WithField("trace.id", span.SpanContext().TraceID().String()).WithField("span.id", span.SpanContext().SpanID().String())
	defer span.End()

	c.mu.Lock()
	handlersCopy := make(map[string]handler.Handler, len(c.handlers))
	for k, v := range c.handlers {
		handlersCopy[k] = v
	}
	c.mu.Unlock()

	var handlerWg sync.WaitGroup
	var hadError atomic.Bool
	for id, h := range handlersCopy {
		handle := h
		handleId := id
		handlerWg.Add(1)
		routine.Go(handlerLogger, wctx, func(_ context.Context) {
			defer handlerWg.Done()
			cont, handlerErr := c.safeHandle(handle, handlerLogger, wctx, msg)
			if !cont {
				c.mu.Lock()
				delete(c.handlers, handleId)
				c.mu.Unlock()
			}
			if handlerErr != nil {
				hadError.Store(true)
				handlerLogger.WithError(handlerErr).Errorf("Handler [%s] failed.", handleId)
			}
		})
	}
	handlerWg.Wait()
	return !hadError.Load()
}

// safeHandle wraps handler execution with panic recovery.
func (c *Consumer) safeHandle(h handler.Handler, l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (cont bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			cont = true
			err = fmt.Errorf("handler panicked: %v", r)
		}
	}()
	return h(l, ctx, msg)
}
