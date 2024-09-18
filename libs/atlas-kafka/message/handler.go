package message

import (
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Validator[M any] func(m M) bool

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
		validator:  func(m M) bool { return true },
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
			return true, err
		}

		process := config.validator(m)
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
