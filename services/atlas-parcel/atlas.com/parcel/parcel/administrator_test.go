package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// TestUpdateStatusIfPending covers the reviewer's "Important" finding:
// UpdateStatusIfPending must be a compare-and-swap on status='pending', not
// an unconditional write. A row that is already resolved (e.g. StatusReceived)
// must be left untouched and report 0 rows affected — under the plain,
// unpredicated UpdateStatus this same call would succeed and clobber the
// row regardless of its current status, which is exactly the race
// Processor.resolve relies on this predicate to close.
func TestUpdateStatusIfPending(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	earlierResolve := now.Add(-time.Minute)

	tests := []struct {
		name           string
		initialStatus  string
		initialResolve *time.Time
		newStatus      string
		wantRows       int64
		wantStatus     string
		wantResolve    *time.Time
	}{
		{
			name:          "pending row is updated",
			initialStatus: StatusPending,
			newStatus:     StatusReceived,
			wantRows:      1,
			wantStatus:    StatusReceived,
		},
		{
			name:           "already-resolved row is left untouched",
			initialStatus:  StatusReceived,
			initialResolve: &earlierResolve,
			newStatus:      StatusDiscarded,
			wantRows:       0,
			wantStatus:     StatusReceived,
			wantResolve:    &earlierResolve,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := databasetest.NewInMemoryTenantDB(t, Migration)
			tid := uuid.New()
			id := uuid.New()
			ctx := databasetest.TenantContext(tid)

			require.NoError(t, db.WithContext(ctx).Create(&Entity{
				Id: id, WorldId: 0,
				SenderId: 200, SenderAccountId: 1, SenderName: "A",
				RecipientId: 100, RecipientAccountId: 2,
				Status: tt.initialStatus, CreatedAt: now,
				ReceivableAt: now, ExpiresAt: now.Add(ExpiryWindow),
				ResolvedAt: tt.initialResolve,
			}).Error)

			rows, err := UpdateStatusIfPending(db.WithContext(ctx))(id, tt.newStatus, now)
			require.NoError(t, err)
			assert.EqualValues(t, tt.wantRows, rows)

			m, err := ById(id)(db.WithContext(ctx))()
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, m.Status())
			if tt.wantResolve != nil {
				require.NotNil(t, m.ResolvedAt())
				assert.True(t, m.ResolvedAt().Equal(*tt.wantResolve))
			}
		})
	}
}
