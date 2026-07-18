package consumer

import (
	"context"
	"errors"
	"fmt"
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

// errFetchWedged is returned from runFetchLoop when FetchMessage has hit
// its deadline maxConsecutiveTimeouts times in a row without a successful
// fetch in between. The outer start loop treats it identically to any
// other recreate-eligible error: close reader, backoff, rebuild.
var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")

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
	handlers      map[string]handler.Handler
	headerParsers []HeaderParser
	mu            sync.Mutex

	// Read-only after construction; copied from Config in AddConsumer.
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int
	maxInFlight            int

	// Observable state — protected by mu.
	aliveSince          time.Time
	lastFetchAt         time.Time
	lastErrorAt         time.Time
	lastError           string
	recreateCount       int
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
	idleTicks           int
	lastIdleTickAt      time.Time
	noProgressTicks     int
	lastNoProgressAt    time.Time

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

// Snapshot is a point-in-time view of a Consumer's observable state, suitable
// for JSON serialization by the debug route.
type Snapshot struct {
	Name                string
	Topic               string
	GroupID             string
	Brokers             []string
	AliveSince          time.Time
	LastFetchAt         time.Time
	LastErrorAt         time.Time
	LastError           string
	RecreateCount       int
	HandlerCount        int
	LastTimeoutAt       time.Time
	ConsecutiveTimeouts int
	IdleTicks           int
	LastIdleTickAt      time.Time
	NoProgressTicks     int
	LastNoProgressAt    time.Time
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
	brokers := append([]string(nil), c.brokers...)
	return Snapshot{
		Name:                c.name,
		Topic:               c.topic,
		GroupID:             c.groupId,
		Brokers:             brokers,
		AliveSince:          c.aliveSince,
		LastFetchAt:         c.lastFetchAt,
		LastErrorAt:         c.lastErrorAt,
		LastError:           c.lastError,
		RecreateCount:       c.recreateCount,
		HandlerCount:        len(c.handlers),
		LastTimeoutAt:       c.lastTimeoutAt,
		ConsecutiveTimeouts: c.consecutiveTimeouts,
		IdleTicks:           c.idleTicks,
		LastIdleTickAt:      c.lastIdleTickAt,
		NoProgressTicks:     c.noProgressTicks,
		LastNoProgressAt:    c.lastNoProgressAt,
		TimeToFirstFetch:    c.timeToFirstFetch,
		LastFetchDuration:   c.lastFetchDuration,
		MaxFetchDuration:    c.maxFetchDuration,
		LastHandlerDuration: c.lastHandlerDuration,
		MaxHandlerDuration:  c.maxHandlerDuration,
		TotalBackoff:        c.totalBackoff,
	}
}

func (c *Consumer) onReaderCreated(attempt int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliveSince = time.Now()
	c.readerCreatedAt = time.Now()
	c.awaitingFirstFetch = true
	if attempt > 0 {
		c.recreateCount++
		c.lastError = ""
		c.consecutiveTimeouts = 0
		c.lastTimeoutAt = time.Time{}
	}
}

func (c *Consumer) recordFetch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastFetchAt = now
	c.lastError = ""
	c.consecutiveTimeouts = 0
	if c.awaitingFirstFetch {
		c.timeToFirstFetch = now.Sub(c.readerCreatedAt)
		c.awaitingFirstFetch = false
	}
}

// recordIdleTick marks one deadline expiration on a reader that is still
// making fetch attempts. Idle is healthy: it resets the no-progress
// escalation counter and touches no error state.
func (c *Consumer) recordIdleTick() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idleTicks++
	c.lastIdleTickAt = time.Now()
	c.consecutiveTimeouts = 0
}

// recordNoProgressTick marks one deadline expiration with zero reader
// progress — a stall suspect. Returns the new consecutive count so callers
// can branch on the threshold without a second mutex acquisition.
func (c *Consumer) recordNoProgressTick() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastTimeoutAt = now
	c.lastNoProgressAt = now
	c.noProgressTicks++
	c.consecutiveTimeouts++
	return c.consecutiveTimeouts
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
func (c *Consumer) handleFetchDeadline(l logrus.FieldLogger, reader KafkaReader) error {
	if readerMadeProgress(reader) {
		c.recordIdleTick()
		l.Debugf("Fetch deadline expired on idle topic [%s]; reader healthy, continuing.", c.topic)
		return nil
	}
	consecutive := c.recordNoProgressTick()
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

// start owns the full reader lifecycle: create reader → run fetch loop →
// close reader → backoff → repeat, until the parent context is canceled.
// Only a canceled parent ctx means shutdown; every other error (including
// io.EOF) flows through the backoff + recreate path.
func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer.")

	backoff := newFetchBackoff()
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			l.Infof("Parent context canceled; shutting down topic consumer.")
			return
		}

		reader := c.rp(c.readerConfig)
		c.onReaderCreated(attempt)
		if attempt == 0 {
			l.Infof("Start consuming topic.")
		} else {
			l.Infof("Recreated reader for topic (attempt %d).", attempt)
		}

		err := c.runFetchLoop(l, ctx, reader)
		if cerr := reader.Close(); cerr != nil {
			l.WithError(cerr).Debugf("Error closing reader during recreate.")
		}

		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			l.Infof("Topic consumer stopped.")
			return
		}

		c.recordError(err)
		l.WithError(err).Errorf("Fetcher exited; recreating reader after backoff.")
		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped during backoff.")
			return
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}
}

// runFetchLoop dispatches to the serial or parallel fetch loop depending on
// c.maxInFlight. Default (maxInFlight == 1) uses the serial path, which is
// bit-exact with the original implementation.
func (c *Consumer) runFetchLoop(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	if c.maxInFlight <= 1 {
		return c.runFetchLoopSerial(l, ctx, reader)
	}
	return c.runFetchLoopParallel(l, ctx, reader)
}

// runFetchLoopSerial is the original single-goroutine fetch loop. It blocks
// until the reader errors or ctx is canceled.
//
// Each iteration runs FetchMessage under a per-call deadline
// (c.fetchTimeout) that acts as a liveness tick. An expiration on a reader
// that is still making fetch attempts (per Stats() deltas) is an idle
// tick — healthy, never a recreate. Only ticks with zero reader progress
// count toward c.maxConsecutiveTimeouts; at the threshold the loop returns
// errFetchWedged so the outer start loop closes and recreates the reader.
// A successful fetch resets the counter via recordFetch.
func (c *Consumer) runFetchLoopSerial(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader); werr != nil {
					return werr
				}
				continue
			}
			return err
		}

		c.recordFetch()
		l.Debugf("Message received %s.", string(msg.Value))
		handlerStart := time.Now()
		ok := c.processMessage(l, ctx, msg)
		c.recordHandlerDuration(time.Since(handlerStart))
		if ok {
			if cerr := reader.CommitMessages(ctx, msg); cerr != nil {
				l.WithError(cerr).Warnf("Could not commit message offset, it may be redelivered.")
			}
		}
	}
}

// runFetchLoopParallel is an opt-in parallel fetch loop that uses a
// prefix-commit cursor to commit offsets in order even when handlers complete
// out of order. Up to c.maxInFlight handlers run concurrently; the in-flight
// queue is capped at 4*c.maxInFlight to bound memory growth when the head is
// stuck on a failing message.
//
// Commit semantics: only the highest contiguously-completed offset is
// committed. A failed handler (processMessage returning false) blocks the
// cursor — subsequent messages are not committed until the failed message is
// redelivered and succeeds (matching at-least-once semantics).
func (c *Consumer) runFetchLoopParallel(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	type pending struct {
		msg  kafka.Message
		done atomic.Bool
		ok   atomic.Bool
	}

	maxQueue := 4 * c.maxInFlight
	sem := make(chan struct{}, c.maxInFlight)

	var qmu sync.Mutex // guards queue slice header
	var queue []*pending

	advanceCommit := func() {
		qmu.Lock()
		i := 0
		for i < len(queue) && queue[i].done.Load() && queue[i].ok.Load() {
			i++
		}
		if i == 0 {
			qmu.Unlock()
			return
		}
		commitMsg := queue[i-1].msg
		queue = queue[i:]
		qmu.Unlock()
		if cerr := reader.CommitMessages(ctx, commitMsg); cerr != nil {
			l.WithError(cerr).Warn("Could not commit message offset; may be redelivered.")
		}
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Back-pressure: stop fetching when the queue is full (head stuck on a
		// failure). Wait one fetchTimeout tick then retry; advanceCommit may
		// have moved the cursor by then.
		qmu.Lock()
		full := len(queue) >= maxQueue
		qmu.Unlock()
		if full {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.fetchTimeout):
			}
			advanceCommit()
			continue
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader); werr != nil {
					return werr
				}
				// In-flight goroutines may have completed; try to advance.
				advanceCommit()
				continue
			}
			return err
		}

		c.recordFetch()
		l.Debugf("Message received %s.", string(msg.Value))

		pm := &pending{msg: msg}
		qmu.Lock()
		queue = append(queue, pm)
		qmu.Unlock()

		sem <- struct{}{}
		p := pm
		routine.Go(l, ctx, func(_ context.Context) {
			defer func() { <-sem }()
			handlerStart := time.Now()
			ok := c.processMessage(l, ctx, p.msg)
			c.recordHandlerDuration(time.Since(handlerStart))
			p.ok.Store(ok)
			p.done.Store(true)
			advanceCommit()
		})
	}
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
