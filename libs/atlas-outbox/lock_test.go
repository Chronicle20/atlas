//go:build integration

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
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDrainer_AdvisoryLock_OnlyOneLeader(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dial := postgres.Open(dsn)
	db1, err := gorm.Open(dial, &gorm.Config{})
	require.NoError(t, err)
	db2, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db1))

	require.NoError(t, db1.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	var mu sync.Mutex
	var published []kafka.Message
	pub := outbox.PublisherFunc(func(ctx context.Context, msgs ...kafka.Message) error {
		mu.Lock()
		published = append(published, msgs...)
		mu.Unlock()
		return nil
	})

	d1 := outbox.NewDrainer(logrus.New(), db1, pub, outbox.WithPollInterval(50*time.Millisecond))
	d2 := outbox.NewDrainer(logrus.New(), db2, pub, outbox.WithPollInterval(50*time.Millisecond))
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	go d1.Run(runCtx)
	go d2.Run(runCtx)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(published) >= 1
	}, 1500*time.Millisecond, 25*time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, len(published), "row was published more than once")
}
