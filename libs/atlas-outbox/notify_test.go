//go:build integration

package outbox_test

import (
	"context"
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

func TestDrainer_NotifyAcceleratesPublish(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	pubCh := make(chan time.Time, 1)
	pub := outbox.PublisherFunc(func(ctx context.Context, msgs ...kafka.Message) error {
		pubCh <- time.Now()
		return nil
	})
	d := outbox.NewDrainer(logrus.New(), db, pub,
		outbox.WithPollInterval(2*time.Second),
		outbox.WithDSN(dsn),
	)
	runCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	go d.Run(runCtx)

	time.Sleep(50 * time.Millisecond) // let the leader establish

	insertedAt := time.Now()
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	publishedAt := <-pubCh
	require.Less(t, publishedAt.Sub(insertedAt), 500*time.Millisecond,
		"NOTIFY should wake the leader well before the 2s poll")
}
