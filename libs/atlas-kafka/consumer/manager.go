package consumer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-retry"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type KafkaReader interface {
	MessageReader
	MessageCommitter
	io.Closer
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

		r := m.rp(kafka.ReaderConfig{
			Brokers:     c.brokers,
			Topic:       c.topic,
			GroupID:     c.groupId,
			MaxWait:     c.maxWait,
			StartOffset: c.startOffset,
		})

		con := &Consumer{
			name:          c.name,
			reader:        r,
			handlers:      make(map[string]handler.Handler),
			headerParsers: c.headerParsers,
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

type Consumer struct {
	name          string
	reader        KafkaReader
	handlers      map[string]handler.Handler
	headerParsers []HeaderParser
	mu            sync.Mutex
}

func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer.")

	readerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg kafka.Message
			readerFunc := func(attempt int) (bool, error) {
				var err error
				msg, err = c.reader.FetchMessage(readerCtx)
				if err == io.EOF || errors.Is(err, context.Canceled) {
					return false, err
				} else if err != nil {
					l.WithError(err).Warnf("Could not fetch message on topic, will retry.")
					return true, err
				}
				return false, err
			}

			cfg := retry.DefaultConfig().WithMaxRetries(10).WithInitialDelay(100 * time.Millisecond).WithMaxDelay(10 * time.Second)
			err := retry.Try(readerCtx, cfg, readerFunc)
			if err == io.EOF || errors.Is(err, context.Canceled) {
				l.Infof("Reader closed, shutdown.")
				return
			} else if err != nil {
				l.WithError(err).Errorf("Could not successfully fetch message, exiting consumer loop.")
				return
			}

			l.Debugf("Message received %s.", string(msg.Value))
			if c.processMessage(l, readerCtx, msg) {
				if err = c.reader.CommitMessages(readerCtx, msg); err != nil {
					l.WithError(err).Warnf("Could not commit message offset, it may be redelivered.")
				}
			}
		}
	}()

	l.Infof("Start consuming topic.")
	<-ctx.Done()
	l.Infof("Shutting down topic consumer.")
	if err := c.reader.Close(); err != nil {
		l.WithError(err).Errorf("Error closing reader.")
	}
	<-done
	l.Infof("Topic consumer stopped.")
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
