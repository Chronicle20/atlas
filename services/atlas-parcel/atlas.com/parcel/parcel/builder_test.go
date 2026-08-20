package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestBuilderRoundTrip is table-driven over run: each row exercises a
// distinct Builder chain (or Make) with a distinct assertion set, so the
// scenario itself is carried as a closure rather than shared data fields.
func TestBuilderRoundTrip(t *testing.T) {
	itemId := uint32(1302000)
	now := time.Now().UTC().Truncate(time.Second)
	resolvedAt := now.Add(time.Hour)
	lastNotified := now.Add(2 * time.Hour)
	snapshot := AssetData{Quantity: 1, OwnerId: 1000}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "full parcel", run: func(t *testing.T) {
			id := uuid.New()
			m, err := NewBuilder().
				SetId(id).
				SetWorldId(world.Id(1)).
				SetSenderId(100).
				SetSenderAccountId(200).
				SetSenderName("Bob").
				SetRecipientId(300).
				SetRecipientAccountId(400).
				SetMessage("Here you go").
				SetMesoAmount(5000).
				SetFeePaid(100).
				SetItemId(&itemId).
				SetItemType(1).
				SetQuantity(1).
				SetItemSnapshot(snapshot).
				SetStatus(StatusPending).
				SetQuick(true).
				SetReturned(false).
				SetCreatedAt(now).
				SetReceivableAt(now.Add(ReceivableDelay)).
				SetExpiresAt(now.Add(ExpiryWindow)).
				SetResolvedAt(&resolvedAt).
				SetLastNotified(&lastNotified).
				Build()
			require.NoError(t, err)

			assert.Equal(t, id, m.Id())
			assert.Equal(t, world.Id(1), m.WorldId())
			assert.Equal(t, uint32(100), m.SenderId())
			assert.Equal(t, uint32(200), m.SenderAccountId())
			assert.Equal(t, "Bob", m.SenderName())
			assert.Equal(t, uint32(300), m.RecipientId())
			assert.Equal(t, uint32(400), m.RecipientAccountId())
			assert.Equal(t, "Here you go", m.Message())
			assert.Equal(t, uint32(5000), m.MesoAmount())
			assert.Equal(t, uint32(100), m.FeePaid())
			require.NotNil(t, m.ItemId())
			assert.Equal(t, itemId, *m.ItemId())
			assert.Equal(t, byte(1), m.ItemType())
			assert.Equal(t, uint16(1), m.Quantity())
			assert.Equal(t, snapshot, m.ItemSnapshot())
			assert.Equal(t, StatusPending, m.Status())
			assert.True(t, m.Quick())
			assert.False(t, m.Returned())
			assert.Equal(t, now, m.CreatedAt())
			assert.Equal(t, now.Add(ReceivableDelay), m.ReceivableAt())
			assert.Equal(t, now.Add(ExpiryWindow), m.ExpiresAt())
			require.NotNil(t, m.ResolvedAt())
			assert.Equal(t, resolvedAt, *m.ResolvedAt())
			require.NotNil(t, m.LastNotified())
			assert.Equal(t, lastNotified, *m.LastNotified())
		}},
		{name: "meso only", run: func(t *testing.T) {
			m, err := NewBuilder().
				SetId(uuid.New()).
				SetSenderId(100).
				SetRecipientId(300).
				SetMesoAmount(5000).
				SetStatus(StatusPending).
				Build()
			require.NoError(t, err)
			assert.Nil(t, m.ItemId())
			assert.Equal(t, uint32(5000), m.MesoAmount())
		}},
		{name: "return leg", run: func(t *testing.T) {
			m, err := NewBuilder().
				SetId(uuid.New()).
				SetSenderId(100).
				SetRecipientId(300).
				SetStatus(StatusPending).
				SetReturned(true).
				SetFeePaid(0).
				SetCreatedAt(now).
				SetReceivableAt(now).
				Build()
			require.NoError(t, err)
			assert.True(t, m.Returned())
			assert.Equal(t, uint32(0), m.FeePaid())
			assert.Equal(t, now, m.ReceivableAt())
		}},
		{name: "make from entity", run: func(t *testing.T) {
			e := Entity{
				Id:                 uuid.New(),
				TenantId:           uuid.New(),
				WorldId:            2,
				SenderId:           100,
				SenderAccountId:    200,
				SenderName:         "Bob",
				RecipientId:        300,
				RecipientAccountId: 400,
				Message:            "Hi",
				MesoAmount:         5000,
				FeePaid:            100,
				ItemId:             &itemId,
				ItemType:           1,
				Quantity:           1,
				ItemSnapshot:       snapshot,
				Status:             StatusPending,
				Quick:              true,
				Returned:           false,
				CreatedAt:          now,
				ReceivableAt:       now.Add(ReceivableDelay),
				ExpiresAt:          now.Add(ExpiryWindow),
				ResolvedAt:         &resolvedAt,
				LastNotified:       &lastNotified,
			}

			m, err := Make(e)
			require.NoError(t, err)
			assert.Equal(t, e.Id, m.Id())
			assert.Equal(t, world.Id(e.WorldId), m.WorldId())
			assert.Equal(t, e.SenderId, m.SenderId())
			assert.Equal(t, e.SenderAccountId, m.SenderAccountId())
			assert.Equal(t, e.SenderName, m.SenderName())
			assert.Equal(t, e.RecipientId, m.RecipientId())
			assert.Equal(t, e.RecipientAccountId, m.RecipientAccountId())
			assert.Equal(t, e.Message, m.Message())
			assert.Equal(t, e.MesoAmount, m.MesoAmount())
			assert.Equal(t, e.FeePaid, m.FeePaid())
			require.NotNil(t, m.ItemId())
			assert.Equal(t, *e.ItemId, *m.ItemId())
			assert.Equal(t, e.ItemType, m.ItemType())
			assert.Equal(t, e.Quantity, m.Quantity())
			assert.Equal(t, e.ItemSnapshot, m.ItemSnapshot())
			assert.Equal(t, e.Status, m.Status())
			assert.Equal(t, e.Quick, m.Quick())
			assert.Equal(t, e.Returned, m.Returned())
			assert.Equal(t, e.CreatedAt, m.CreatedAt())
			assert.Equal(t, e.ReceivableAt, m.ReceivableAt())
			assert.Equal(t, e.ExpiresAt, m.ExpiresAt())
			require.NotNil(t, m.ResolvedAt())
			assert.Equal(t, *e.ResolvedAt, *m.ResolvedAt())
			require.NotNil(t, m.LastNotified())
			assert.Equal(t, *e.LastNotified, *m.LastNotified())
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
