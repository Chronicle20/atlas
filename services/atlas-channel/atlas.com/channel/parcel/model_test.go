package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	packettest "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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

// TestToPacketItemHasNoSlotPrefix pins the fix for bug-duey-receive-list-
// item-slot-desync: GW_ItemSlotBase::Decode @0x4E33F9 (GMS v83) reads the
// item TYPE byte first — `v2 = CInPacket::Decode1(a2);
// GW_ItemSlotBase::CreateItem(&v6, v2)` — with no leading slot byte, unlike
// the inventory/storage call sites that read a slot before invoking Decode.
// A Model whose attached item carries a non-zero Builder slot must still
// encode onto the wire with the type byte first, or the client desyncs the
// whole Duey receive list. See docs/tasks/task-241-duey-parcel-delivery/
// bug-duey-receive-list-item-slot-desync.md.
func TestToPacketItemHasNoSlotPrefix(t *testing.T) {
	ctx := packettest.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()

	t.Run("stackable", func(t *testing.T) {
		templateId := uint32(2000004)
		m, err := NewBuilder().
			SetId(uuid.New()).
			SetSenderId(1).
			SetSenderName("Alice").
			SetRecipientId(2).
			SetStatus("pending").
			SetExpiresAt(time.Now().Add(29 * 24 * time.Hour)).
			SetItemId(&templateId).
			SetItemType(byte(inventory.TypeValueUse)).
			SetQuantity(5).
			Build()
		require.NoError(t, err)

		p := m.ToPacket()
		asset, ok := p.Item()
		require.True(t, ok, "ToPacket must attach the item to the wire Parcel")

		encoded := asset.Encode(l, ctx)(nil)
		require.NotEmpty(t, encoded)
		assert.Equal(t, byte(2), encoded[0], "stackable item's first wire byte must be the TYPE byte (2), not a slot byte — got % x", encoded[:4])
	})

	t.Run("equip", func(t *testing.T) {
		templateId := uint32(1302000)
		m, err := NewBuilder().
			SetId(uuid.New()).
			SetSenderId(1).
			SetSenderName("Alice").
			SetRecipientId(2).
			SetStatus("pending").
			SetExpiresAt(time.Now().Add(29 * 24 * time.Hour)).
			SetItemId(&templateId).
			SetItemType(byte(inventory.TypeValueEquip)).
			Build()
		require.NoError(t, err)

		p := m.ToPacket()
		asset, ok := p.Item()
		require.True(t, ok, "ToPacket must attach the item to the wire Parcel")

		encoded := asset.Encode(l, ctx)(nil)
		require.NotEmpty(t, encoded)
		assert.Equal(t, byte(1), encoded[0], "equip item's first wire byte must be the TYPE byte (1), not a slot byte — got % x", encoded[:4])
	})
}
