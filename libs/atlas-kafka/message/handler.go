package message

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// previewMax bounds the byte prefix of a malformed Kafka payload that we
// include in the error log. Sized to capture the leading envelope of a typical
// JSON event without risking oversized log lines or sensitive-data exposure.
const previewMax = 256

type Validator[M any] func(l logrus.FieldLogger, ctx context.Context, m M) bool

type Handler[M any] func(l logrus.FieldLogger, ctx context.Context, m M)

type Config[M any] struct {
	persistent bool
	validator  Validator[M]
	handler    Handler[M]
}

//goland:noinspection GoUnusedExportedFunction
func PersistentConfig[M any](handler Handler[M]) Config[M] {
	return Config[M]{
		persistent: true,
		validator:  func(l logrus.FieldLogger, ctx context.Context, m M) bool { return true },
		handler:    handler,
	}
}

//goland:noinspection GoUnusedExportedFunction
func OneTimeConfig[M any](validator Validator[M], handler Handler[M]) Config[M] {
	return Config[M]{
		persistent: false,
		validator:  validator,
		handler:    handler,
	}
}

//goland:noinspection GoUnusedExportedFunction
func AdaptHandler[M any](config Config[M]) handler.Handler {
	h := func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		tem := model.Map[kafka.Message, M](adapt[M])(model.FixedProvider(msg))
		m, err := tem()
		if err != nil {
			preview := msg.Value
			if len(preview) > previewMax {
				preview = preview[:previewMax]
			}
			l.WithFields(logrus.Fields{
				"topic":           msg.Topic,
				"partition":       msg.Partition,
				"offset":          msg.Offset,
				"payload_size":    len(msg.Value),
				"payload_preview": fmt.Sprintf("%q", preview),
				"message_type":    fmt.Sprintf("%T", *new(M)),
			}).WithError(err).Errorf("Failed to unmarshal Kafka message; offset will be committed and the message dropped.")
			return true, nil
		}

		process := config.validator(l, ctx, m)
		if !process {
			return true, nil
		}

		config.handler(l, ctx, m)
		return config.persistent, nil
	}
	return h
}

func adapt[M any](msg kafka.Message) (M, error) {
	var event M
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return event, err
	}
	return event, nil
}
