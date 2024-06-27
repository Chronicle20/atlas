package consumer

import (
	"atlas-kafka/retry"
	"context"
	"errors"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
)

//goland:noinspection GoUnusedExportedFunction
func CreateConsumers(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup, configs ...Config) {
	for _, c := range configs {
		go createConsumer(l, ctx, wg, c)
	}
}

func createConsumer(cl *logrus.Logger, ctx context.Context, wg *sync.WaitGroup, c Config) {
	l := cl.WithFields(logrus.Fields{"originator": c.topic, "type": "kafka_consumer"})

	l.Infof("Creating topic consumer.")

	wg.Add(1)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("BOOTSTRAP_SERVERS")},
		Topic:   c.topic,
		GroupID: c.groupId,
		MaxWait: c.maxWait,
	})

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for {
			var msg kafka.Message
			readerFunc := func(attempt int) (bool, error) {
				var err error
				msg, err = r.ReadMessage(ctx)
				if err == io.EOF || errors.Is(err, context.Canceled) {
					return false, err
				} else if err != nil {
					l.WithError(err).Warnf("Could not read message on topic %s, will retry.", r.Config().Topic)
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
					defer span.Finish()

					c.handler(l, span, msg)
				}()
			}
		}
	}()

	l.Infof("Start consuming topic %s.", c.topic)
	<-ctx.Done()
	l.Infof("Shutting down topic %s consumer.", c.topic)
	if err := r.Close(); err != nil {
		l.WithError(err).Errorf("Error closing reader.")
	}
	wg.Done()
	l.Infof("Topic consumer stopped.")
}
