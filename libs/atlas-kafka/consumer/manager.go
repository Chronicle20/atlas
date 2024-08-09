package consumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/retry"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"io"
	"sync"
)

type Manager struct {
	mu        *sync.Mutex
	consumers map[string]*Consumer
}

var manager *Manager
var once sync.Once

//goland:noinspection GoUnusedExportedFunction
func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			mu:        &sync.Mutex{},
			consumers: make(map[string]*Consumer),
		}
	})
	return manager
}

func (m *Manager) AddConsumer(cl logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config Config) {
	return func(config Config) {
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, exists := m.consumers[config.topic]; exists {
			cl.Infof("Consumer for topic [%s] is already registered.", config.topic)
			return
		}

		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers: config.brokers,
			Topic:   config.topic,
			GroupID: config.groupId,
			MaxWait: config.maxWait,
		})

		c := &Consumer{
			name:     config.name,
			reader:   r,
			handlers: make(map[string]handler.Handler),
		}

		m.consumers[config.topic] = c

		l := cl.WithFields(logrus.Fields{"originator": config.topic, "type": "kafka_consumer"})
		go c.start(l, ctx, wg)
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
	name     string
	reader   *kafka.Reader
	handlers map[string]handler.Handler
	mu       sync.Mutex
}

func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer.")

	go func() {
		ctx, cancel := context.WithCancel(ctx)
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
					headers := make(map[string]string)
					for _, header := range msg.Headers {
						headers[header.Key] = string(header.Value)
					}

					spanContext, _ := opentracing.GlobalTracer().Extract(opentracing.TextMap, opentracing.TextMapCarrier(headers))
					span := opentracing.StartSpan(c.name, opentracing.FollowsFrom(spanContext))
					sl := l.WithField("span.id", fmt.Sprintf("%v", span))
					defer span.Finish()

					c.mu.Lock()
					handlers := c.handlers
					c.mu.Unlock()

					for id, h := range handlers {
						var handle = h
						var handleId = id
						go func() {
							var cont bool
							cont, err = handle(sl, span, msg)
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
								sl.WithError(err).Errorf("Handler [%s] failed.", handleId)
							}
						}()
					}
					c.mu.Unlock()
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
