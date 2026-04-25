package producer

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-retry"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
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

				cfg := retry.DefaultConfig().WithMaxRetries(10).WithInitialDelay(100 * time.Millisecond).WithMaxDelay(10 * time.Second)
				for _, m := range ms {
					err = retry.Try(context.Background(), cfg, tryMessage(l, w)(m))
					if err != nil {
						l.WithError(err).Errorf("Unable to emit event on topic [%s].", w.Topic())
						return err
					}
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
