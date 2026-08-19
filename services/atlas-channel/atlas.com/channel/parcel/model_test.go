package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToPacketExpiresAt pins task-23's RISK-4 fix: the client's receive
// guard (CTabReceive::ReceiveParcel, v72 @0x65AF41 / v83 @0x6F0D11) divides
// UNSIGNED, so the wire's +21 field must be a FUTURE deadline — ExpiresAt —
// never CreatedAt, which is always in the past. See docs/tasks/
// task-241-duey-parcel-delivery/context.md §11.
func TestToPacketExpiresAt(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expires := created.Add(29 * 24 * time.Hour)

	rm := RestModel{
		Id:         uuid.New().String(),
		SenderName: "Alice",
		MesoAmount: 1000,
		CreatedAt:  created,
		ExpiresAt:  expires,
	}
	m, err := Extract(rm)
	require.NoError(t, err)

	p := m.ToPacket()
	assert.True(t, p.ExpiresAt().Equal(expires), "PARCEL wire +21 must carry ExpiresAt (a future deadline); got %v want %v", p.ExpiresAt(), expires)
	assert.False(t, p.ExpiresAt().Equal(created), "PARCEL wire +21 must NOT carry CreatedAt (always in the past) — this is the defect RISK-4 fixes")
}
