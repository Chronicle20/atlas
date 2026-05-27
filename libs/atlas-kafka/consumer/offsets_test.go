//go:build integration

package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestReadEndOffsets_ReturnsCurrentEndPerPartition(t *testing.T) {
	ctx := context.Background()
	kc, err := tckafka.Run(ctx, "confluentinc/cp-kafka:7.6.0", tckafka.WithClusterID("atlas-test"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = kc.Terminate(ctx) })

	brokers, err := kc.Brokers(ctx)
	require.NoError(t, err)

	const topic = "T"

	conn, err := (&kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}).
		DialContext(ctx, "tcp", brokers[0])
	require.NoError(t, err)
	require.NoError(t, conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))
	_ = conn.Close()

	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: false,
		BatchTimeout:           50 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
	}
	defer w.Close()

	require.NoError(t, w.WriteMessages(ctx,
		kafka.Message{Key: []byte("k1"), Value: []byte("v1")},
		kafka.Message{Key: []byte("k2"), Value: []byte("v2")},
		kafka.Message{Key: []byte("k3"), Value: []byte("v3")},
	))

	require.Eventually(t, func() bool {
		got, err := consumer.ReadEndOffsets(ctx, brokers, topic)
		if err != nil {
			return false
		}
		return got[0] == 3
	}, 5*time.Second, 100*time.Millisecond, "expected end offset 3 on partition 0")
}

func TestReadEndOffsets_EmptyBrokersErrors(t *testing.T) {
	_, err := consumer.ReadEndOffsets(context.Background(), nil, "T")
	require.Error(t, err)
}
