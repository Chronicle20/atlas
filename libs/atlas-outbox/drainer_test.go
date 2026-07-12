package outbox_test

import (
	"context"
	"sync"
	"testing"
	"time"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakePublisher struct {
	mu       sync.Mutex
	messages []kafka.Message
}

func (f *fakePublisher) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msgs...)
	return nil
}

func TestDrainer_PublishesUnsentRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	pub := &fakePublisher{}
	l := logrus.New()
	d := outbox.NewDrainer(l, db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond),
		outbox.WithBatchSize(10),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.NotNil(t, ent.SentAt)
}

func TestDrainer_ReattachesHeaders(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	// Binary tenant-version value: NUL + 0xB9 (v185) — must survive byte-exact.
	binVal := string([]byte{0x00, 0xB9})
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{
			Topic: "T", Key: []byte("k"), Value: []byte("v"),
			Headers: map[string]string{"TENANT_ID": "abc", "MAJOR_VERSION": binVal},
		})
	}))

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	got := map[string][]byte{}
	for _, h := range pub.messages[0].Headers {
		got[h.Key] = h.Value
	}
	require.Equal(t, []byte("abc"), got["TENANT_ID"])
	require.Equal(t, []byte{0x00, 0xB9}, got["MAJOR_VERSION"])
}

func TestDrainer_EmptyHeadersPublishNoHeaderSlice(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)
	require.Empty(t, pub.messages[0].Headers)
}

func TestDrainer_PublishesInIdOrderOnTimestampTie(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	// Same enqueued_at on every row — the Postgres intra-transaction reality.
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 3; i++ {
		require.NoError(t, db.Create(&outbox.Entity{
			Topic: "T", MessageKey: []byte("k"),
			MessageValue: []byte{byte(i)}, EnqueuedAt: ts,
		}).Error)
	}

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 3
	}, 400*time.Millisecond, 10*time.Millisecond)
	require.Equal(t, []byte{1}, pub.messages[0].Value)
	require.Equal(t, []byte{2}, pub.messages[1].Value)
	require.Equal(t, []byte{3}, pub.messages[2].Value)
}
