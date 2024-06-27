package producer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/Chronicle20/atlas-kafka/retry"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

//goland:noinspection GoUnusedExportedFunction
func CreateKey(key int) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b, uint32(key))
	return b
}

//goland:noinspection GoUnusedExportedFunction
func ProduceEvent(l logrus.FieldLogger, span opentracing.Span, topic string) func(key []byte, event interface{}) {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS")),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}

	return func(key []byte, event interface{}) {
		value, err := json.Marshal(event)
		l.WithField("message", string(value)).Debugf("Writing message to topic %s.", topic)
		if err != nil {
			l.WithError(err).Fatalf("Unable to marshall event for topic %s.", topic)
		}

		writeMessage := func(attempt int) (bool, error) {
			m := kafka.Message{Key: key, Value: value}
			headers := make(map[string]string)
			err = opentracing.GlobalTracer().Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(headers))
			if err != nil {
				l.WithError(err).Warnf("Unable to inject OpenTracing information.")
				return false, err
			}
			for k, v := range headers {
				m.Headers = append(m.Headers, kafka.Header{Key: k, Value: []byte(v)})
			}

			err = w.WriteMessages(context.Background(), m)
			if err != nil {
				l.WithError(err).Warnf("Unable to emit event on topic %s, will retry.", topic)
				return true, err
			}
			return false, err
		}

		err = retry.Try(writeMessage, 10)
		if err != nil {
			l.WithError(err).Fatalf("Unable to emit event on topic %s.", topic)
		}
	}
}
