package outbox_test

import (
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackfill_Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	type src struct{ ID, Body string }
	srcs := []src{{"a", `{"x":1}`}, {"b", `{"x":2}`}}

	keyFn := func(v any) ([]byte, error) { return []byte(v.(src).ID), nil }
	valFn := func(v any) ([]byte, error) { return []byte(v.(src).Body), nil }
	loader := func() ([]any, error) {
		out := make([]any, len(srcs))
		for i, s := range srcs {
			out[i] = s
		}
		return out, nil
	}

	n, err := outbox.Backfill(db, "T", loader, keyFn, valFn)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = outbox.Backfill(db, "T", loader, keyFn, valFn)
	require.NoError(t, err)
	require.Equal(t, 0, n, "second backfill must be no-op")

	var rows []outbox.Entity
	require.NoError(t, db.Order("id ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, []byte("a"), rows[0].MessageKey)
	require.Equal(t, []byte("b"), rows[1].MessageKey)
}
