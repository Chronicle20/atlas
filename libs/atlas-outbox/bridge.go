package outbox

import (
	"context"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnqueueBuffer persists a message.Buffer-shaped payload (env-var token →
// messages) as outbox rows inside tx. Tokens are resolved to real topic
// names via topic.EnvProvider; span + tenant headers are derived from ctx
// exactly as the direct producer path derives them at emit time. Message
// key and value bytes pass through unchanged. Any failure returns an
// error, failing the enclosing transaction.
func EnqueueBuffer(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB, contents map[string][]kafka.Message) error {
	headers, err := headerMap(ctx)
	if err != nil {
		return err
	}
	for token, msgs := range contents {
		t, err := topic.EnvProvider(l)(token)()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := Enqueue(tx, Message{Topic: t, Key: m.Key, Value: m.Value, Headers: headers}); err != nil {
				return err
			}
		}
	}
	return nil
}

// headerMap merges the span and tenant decorators into one map — the same
// key set the direct path's produceHeaders folds (span and tenant key sets
// are disjoint, so map-merge is equivalent to the append-fold).
func headerMap(ctx context.Context) (map[string]string, error) {
	headers := make(map[string]string)
	decorators := []kafkaproducer.HeaderDecorator{
		kafkaproducer.SpanHeaderDecorator(ctx),
		kafkaproducer.TenantHeaderDecorator(ctx),
	}
	for _, d := range decorators {
		hm, err := d()
		if err != nil {
			return nil, err
		}
		for k, v := range hm {
			headers[k] = v
		}
	}
	return headers, nil
}
