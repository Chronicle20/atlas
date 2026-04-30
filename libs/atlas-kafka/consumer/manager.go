package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-retry"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
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

var manager *Manager
var once sync.Once

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

		readerConfig := kafka.ReaderConfig{
			Brokers:     c.brokers,
			Topic:       c.topic,
			GroupID:     c.groupId,
			MaxWait:     c.maxWait,
			StartOffset: c.startOffset,
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
		}

		m.consumers[c.topic] = con

		l := cl.WithFields(logrus.Fields{"originator": c.topic, "type": "kafka_consumer"})
		go con.start(l, ctx, wg)
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

	// Observable state — protected by mu.
	aliveSince          time.Time
	lastFetchAt         time.Time
	lastErrorAt         time.Time
	lastError           string
	recreateCount       int
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
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
	}
}

func (c *Consumer) onReaderCreated(attempt int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliveSince = time.Now()
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
	c.lastFetchAt = time.Now()
	c.lastError = ""
	c.consecutiveTimeouts = 0
}

// recordTimeout marks one deadline expiration; called per tick by runFetchLoop.
// Idle, not an error: lastError / lastErrorAt are untouched.
func (c *Consumer) recordTimeout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastTimeoutAt = time.Now()
	c.consecutiveTimeouts++
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
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped during backoff.")
			return
		case <-time.After(backoff.next()):
		}
	}
}

// runFetchLoop blocks the caller until the supplied reader errors out or
// the parent ctx is canceled. On return, the reader should be closed by the
// caller and (if the error is not a ctx-cancel) a new reader should be
// created. The inner retry is intentionally short — a transient kafka-go
// hiccup that self-resolves within ~1s stays on the current reader;
// everything else falls through to reader recreation.
func (c *Consumer) runFetchLoop(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var msg kafka.Message
		cfg := retry.DefaultConfig().
			WithMaxRetries(3).
			WithInitialDelay(100 * time.Millisecond).
			WithMaxDelay(500 * time.Millisecond)
		err := retry.Try(ctx, cfg, func(attempt int) (bool, error) {
			var ferr error
			msg, ferr = reader.FetchMessage(ctx)
			if ferr == nil {
				return false, nil
			}
			if errors.Is(ferr, context.Canceled) {
				return false, ferr
			}
			l.WithError(ferr).Warnf("Could not fetch message on topic, will retry.")
			return true, ferr
		})
		if err != nil {
			return err
		}

		c.recordFetch()
		l.Debugf("Message received %s.", string(msg.Value))
		if c.processMessage(l, ctx, msg) {
			if cerr := reader.CommitMessages(ctx, msg); cerr != nil {
				l.WithError(cerr).Warnf("Could not commit message offset, it may be redelivered.")
			}
		}
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
		var handle = h
		var handleId = id
		handlerWg.Add(1)
		go func() {
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
		}()
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
