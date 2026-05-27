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
