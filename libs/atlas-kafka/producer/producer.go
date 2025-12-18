package producer

import (
	"context"
	"encoding/binary"
	"github.com/Chronicle20/atlas-kafka/retry"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
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

type Writer interface {
	Topic() string
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type WriterImpl struct {
	w *kafka.Writer
}

func (i WriterImpl) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return i.w.WriteMessages(ctx, msgs...)
}

func (i WriterImpl) Topic() string {
	return i.w.Topic
}

func (i WriterImpl) Close() error {
	return i.w.Close()
}

//goland:noinspection GoUnusedExportedFunction
func WriterProvider(provider topic.Provider) model.Provider[Writer] {
	t, err := provider()
	if err != nil {
		return model.ErrorProvider[Writer](err)
	}
	return func() (Writer, error) {
		w := WriterImpl{
			w: &kafka.Writer{
				Addr:                   kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS")),
				Topic:                  t,
				Balancer:               &kafka.LeastBytes{},
				BatchTimeout:           50 * time.Millisecond,
				AllowAutoTopicCreation: true,
			},
		}
		return w, nil
	}
}

//goland:noinspection GoUnusedExportedFunction
func Produce(l logrus.FieldLogger) func(provider model.Provider[Writer]) func(decorators ...HeaderDecorator) MessageProducer {
	return func(provider model.Provider[Writer]) func(decorators ...HeaderDecorator) MessageProducer {
		return func(decorators ...HeaderDecorator) MessageProducer {
			w, err := provider()
			if err != nil {
				return ErrMessageProducer(err)
			}

			return func(provider model.Provider[[]kafka.Message]) error {
				var ms []kafka.Message
				ms, err = model.SliceMap(DecorateHeaders(decorators...))(provider)()()
				if err != nil {
					return err
				}

				for _, m := range ms {
					err = retry.Try(tryMessage(l, w)(m), 10)
					if err != nil {
						l.WithError(err).Errorf("Unable to emit event on topic [%s].", w.Topic())
						return err
					}
				}

				err = w.Close()
				if err != nil {
					return err
				}

				return nil
			}
		}
	}
}

func DecorateHeaders(decorators ...HeaderDecorator) model.Transformer[kafka.Message, kafka.Message] {
	return func(m kafka.Message) (kafka.Message, error) {
		var err error
		m.Headers, err = produceHeaders(decorators...)
		if err != nil {
			return m, err
		}
		return m, nil
	}
}

func tryMessage(l logrus.FieldLogger, w Writer) func(m kafka.Message) func(attempt int) (bool, error) {
	return func(m kafka.Message) func(attempt int) (bool, error) {
		return func(attempt int) (bool, error) {
			err := w.WriteMessages(context.Background(), m)
			if err != nil {
				l.WithError(err).Warnf("Unable to emit event on topic [%s], will retry.", w.Topic())
				return true, err
			}
			return false, err
		}
	}
}
