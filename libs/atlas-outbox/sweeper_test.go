package outbox_test

import (
	"context"
	"testing"
	"time"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDrainer_SweeperDeletesOldSentRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	old := time.Now().Add(-10 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	require.NoError(t, db.Create(&outbox.Entity{Topic: "T", MessageKey: []byte("k1"), SentAt: &old}).Error)
	require.NoError(t, db.Create(&outbox.Entity{Topic: "T", MessageKey: []byte("k2"), SentAt: &recent}).Error)
	require.NoError(t, db.Create(&outbox.Entity{Topic: "T", MessageKey: []byte("k3")}).Error) // unsent stays

	d := outbox.NewDrainer(logrus.New(), db,
		outbox.PublisherFunc(func(context.Context, ...kafka.Message) error { return nil }),
		outbox.WithRetention(7*24*time.Hour))

	require.NoError(t, d.SweepOnce(context.Background()))
	var count int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	require.Equal(t, int64(2), count, "old sent row deleted; recent sent and unsent rows remain")
}
