package consumer

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/retry"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"io"
	"sync"
)

type KafkaReader interface {
	MessageReader
	io.Closer
}

type MessageReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
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
			Brokers: c.brokers,
			Topic:   c.topic,
			GroupID: c.groupId,
			MaxWait: c.maxWait,
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

	go func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		for {
			var msg kafka.Message
			readerFunc := func(attempt int) (bool, error) {
				var err error
				msg, err = c.reader.ReadMessage(ctx)
				if err == io.EOF || errors.Is(err, context.Canceled) {
					return false, err
				} else if err != nil {
					l.WithError(err).Warnf("Could not read message on topic, will retry.")
					return true, err
				}
				return false, err
			}

			err := retry.Try(readerFunc, 10)
			if err == io.EOF || errors.Is(err, context.Canceled) || len(msg.Value) == 0 {
				l.Infof("Reader closed, shutdown.")
				return
			} else if err != nil {
				l.WithError(err).Errorf("Could not successfully read message.")
			} else {
				l.Debugf("Message received %s.", string(msg.Value))
				go func() {
					wctx := ctx
					for _, p := range c.headerParsers {
						wctx = p(wctx, msg.Headers)
					}

					var span trace.Span
					wctx, span = otel.GetTracerProvider().Tracer("atlas-kafka").Start(wctx, c.name)
					l = l.WithField("trace.id", span.SpanContext().TraceID().String()).WithField("span.id", span.SpanContext().SpanID().String())
					defer span.End()

					c.mu.Lock()
					handlers := c.handlers
					c.mu.Unlock()

					for id, h := range handlers {
						var handle = h
						var handleId = id
						go func() {
							var cont bool
							cont, err = handle(l, wctx, msg)
							if !cont {
								c.mu.Lock()
								var nh = make(map[string]handler.Handler)
								for tid, th := range c.handlers {
									if handleId != tid {
										nh[tid] = th
									}
								}
								c.handlers = nh
								c.mu.Unlock()
							}
							if err != nil {
								l.WithError(err).Errorf("Handler [%s] failed.", handleId)
							}
						}()
					}
				}()
			}
		}
	}()

	l.Infof("Start consuming topic.")
	<-ctx.Done()
	l.Infof("Shutting down topic consumer.")
	if err := c.reader.Close(); err != nil {
		l.WithError(err).Errorf("Error closing reader.")
	}
	l.Infof("Topic consumer stopped.")
}
