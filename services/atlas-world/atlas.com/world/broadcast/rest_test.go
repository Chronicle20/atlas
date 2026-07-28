package broadcast

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransform_IdleQueue(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	q := QueueModel{}

	rm, err := Transform(FamilyTV, q, now)
	require.NoError(t, err)
	require.Equal(t, uint32(0), rm.ActiveRemainingSeconds)
	require.Equal(t, 0, rm.PendingCount)
	require.Equal(t, uint32(0), rm.WaitSeconds)
	require.Equal(t, FamilyTV, rm.Family)
	require.Equal(t, FamilyTV, rm.Id)
}

func TestTransform_BusyQueue(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	active := Entry{
		DurationSeconds: 10,
		ActivatedAt:     now,
		ExpiresAt:       now.Add(10 * time.Second),
	}
	pending := []Entry{
		{DurationSeconds: 15},
		{DurationSeconds: 15},
	}
	q := QueueModel{
		Active:  &active,
		Pending: pending,
	}

	rm, err := Transform(FamilyTV, q, now)
	require.NoError(t, err)
	require.Equal(t, uint32(10), rm.ActiveRemainingSeconds)
	require.Equal(t, 2, rm.PendingCount)
	require.Equal(t, uint32(40), rm.WaitSeconds)
}
