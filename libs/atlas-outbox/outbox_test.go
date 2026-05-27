package outbox_test

import (
	"context"
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnqueue_InsertsRow_InTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	msg := outbox.Message{
		Topic:   "TEST_TOPIC",
		Key:     []byte("k1"),
		Value:   []byte(`{"a":1}`),
		Headers: map[string]string{"trace": "span-1"},
	}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, msg)
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Find(&ents).Error)
	require.Len(t, ents, 1)
	require.Equal(t, "TEST_TOPIC", ents[0].Topic)
	require.Equal(t, []byte("k1"), ents[0].MessageKey)
	require.Equal(t, []byte(`{"a":1}`), ents[0].MessageValue)
	require.Nil(t, ents[0].SentAt)
	_ = context.TODO()
}

func TestEnqueue_TombstoneValueIsNullable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{
			Topic: "TEST_TOPIC",
			Key:   []byte("k1"),
			Value: nil,
		})
	}))

	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.Nil(t, ent.MessageValue)
}
