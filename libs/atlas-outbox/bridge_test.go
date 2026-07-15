package outbox_test

import (
	"context"
	"testing"
	"time"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func bridgeDb(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))
	return db
}

func tenantCtx(t *testing.T) context.Context {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func TestEnqueueBuffer_ResolvesTokenAndPreservesBytes(t *testing.T) {
	db := bridgeDb(t)
	t.Setenv("EVENT_TOPIC_TEST", "real-topic-name")

	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_TEST": {
			{Key: []byte("k1"), Value: []byte(`{"a":1}`)},
			{Key: []byte("k2"), Value: []byte(`{"b":2}`)},
		},
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Order("id ASC").Find(&ents).Error)
	require.Len(t, ents, 2)
	require.Equal(t, "real-topic-name", ents[0].Topic)
	require.Equal(t, []byte("k1"), ents[0].MessageKey)
	require.Equal(t, []byte(`{"a":1}`), ents[0].MessageValue)
	require.Equal(t, []byte("k2"), ents[1].MessageKey)
}

func TestEnqueueBuffer_UnsetTokenFallsThroughToToken(t *testing.T) {
	db := bridgeDb(t)
	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_UNSET": {{Key: []byte("k"), Value: []byte("v")}},
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	}))
	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.Equal(t, "EVENT_TOPIC_UNSET", ent.Topic)
}

// FR-2.2 acceptance: enqueued header set == the direct path's decorator fold.
func TestEnqueueBuffer_HeaderParityWithDirectPath(t *testing.T) {
	db := bridgeDb(t)
	ctx := tenantCtx(t)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), ctx, tx,
			map[string][]kafka.Message{"T": {{Key: []byte("k"), Value: []byte("v")}}})
	}))

	// Direct-path reference: fold the same decorators the service-local
	// ProviderImpl passes to Produce.
	want := map[string][]byte{}
	for _, d := range []kafkaproducer.HeaderDecorator{
		kafkaproducer.SpanHeaderDecorator(ctx),
		kafkaproducer.TenantHeaderDecorator(ctx),
	} {
		hm, err := d()
		require.NoError(t, err)
		for k, v := range hm {
			want[k] = []byte(v)
		}
	}

	// Drain via a short Run window and compare the published header set.
	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx2, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx2)
	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	got := map[string][]byte{}
	for _, h := range pub.messages[0].Headers {
		got[h.Key] = h.Value
	}
	require.Equal(t, want, got)
}

func TestEnqueueBuffer_RowFailureReturnsError(t *testing.T) {
	db := bridgeDb(t)
	contents := map[string][]kafka.Message{
		"T": {{Key: nil, Value: []byte("v")}}, // empty key -> Enqueue rejects
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	})
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	require.Zero(t, count)
}
